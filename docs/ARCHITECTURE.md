# Architecture

```mermaid
flowchart LR
  PSTN((PSTN / SIP trunk)) --> AST[Asterisk<br/>PJSIP + ARI]
  Phones((Agent desk phones /<br/>browser softphone)) <--> AST
  AST <-- "ARI REST + WebSocket events" --> CORE
  AST <-- "ExternalMedia RTP (G.711 μ-law)" --> AI
  subgraph MW [callcenter-middleware (single Go binary)]
    CORE[Call core<br/>single-goroutine actor:<br/>IVR · ACD · voicemail · callbacks ·<br/>transfers · monitoring · outbound]
    AI[AI bridge<br/>RTP ⇄ OpenAI Realtime WS<br/>tools · transcripts]
    API[REST API + WebSocket hub]
    WEB[Embedded React control panel]
    DB[(SQLite)]
    HOOK[Webhook dispatcher]
    CORE <--> DB
    API <--> CORE
    API <--> DB
    CORE -- events --> API
    CORE -- events --> HOOK
    AI -- tool calls --> CORE
  end
  AI <-- "wss realtime" --> OAI[(OpenAI Realtime +<br/>Chat / Transcription)]
  Browser((Admins · supervisors · agents<br/>web panel / Electron app)) <--> API
  Browser --- WEB
  HOOK --> CRM((CRM / helpdesk))
  PROM((Prometheus)) --> API
```

## Call core

`internal/core` owns every live call, queue and agent. All inputs (ARI
events, API commands, AI tool calls and a 1 s timer tick) are funnelled into a
single goroutine, so state transitions never race and need no locks. ARI REST
calls are made from that goroutine with short timeouts.

A call is a set of **legs** (Asterisk channels) with roles:

| Role | Channel | Created by |
|---|---|---|
| caller | inbound trunk channel (or the dialed customer for outbound) | Asterisk `Stasis()` |
| agent | ringing / answered agent phone | `POST /channels` (originate into Stasis) |
| media | ExternalMedia RTP channel for the AI agent | `POST /channels/externalMedia` |
| supervisor / snoop | monitoring supervisor and the snoop on the agent leg | originate + `POST /channels/{id}/snoop` |
| outagent / outcust / external | click-to-call and external transfers | originate |

### Lifecycle

1. `StasisStart` without args → new inbound call; the caller is answered,
   matched to a CRM contact (VIP callers get priority) and an **IVR flow** is
   selected by dialed number (`dnis`, `*` = catch-all). No flow configured →
   the legacy env-driven menu that hands calls to dialplan queues.
2. Schedule closed → after-hours action. `direct_action` skips the menu
   (e.g. AI-first hotlines). Otherwise the greeting plays; DTMF barges in.
3. Actions: `queue`, `ai`, `voicemail`, `callback`, `hangup`, `flow`
   (sub-menu), `agent` (direct), `external` (PSTN number), `dialplan`
   (hand to an Asterisk extension — used by `asterisk` mode queues).
4. **Native ACD**: the caller hears MOH; the dispatcher offers waiting calls
   (queue priority → caller priority → wait time) to available members by
   penalty tier and strategy (`ringall`, `longest_idle`, `round_robin`,
   `fewest_calls`, `random`) with skills filtering. The first agent to answer
   is bridged (optionally recorded); others stop ringing. No answer → next
   agent; repeated misses auto-pause the agent. `max_wait_sec` → overflow
   action; `leave_when_empty`; `*` requests a callback when allowed.
5. Hang-up of either side finishes the call: remaining legs are hung up, the
   agent enters wrap-up, the CDR is written and post-call AI analysis runs.

### Fail-open

- On shutdown, every caller not yet talking to a human is `continue`d to
  `DEFAULT_QUEUE` in `QUEUE_CONTEXT` (Asterisk app_queue); bridged
  conversations stay up in Asterisk.
- The dialplan falls through to the same queue if `Stasis()` cannot start.
- AI failures (no key, OpenAI unreachable, media lost, max duration) hand
  the caller to the profile's handoff queue.
- After an ARI reconnect the core reconciles live calls with
  `GET /channels` and closes calls whose channels vanished.

## AI voice agent

`internal/ai` allocates a UDP port per call and asks Asterisk for an
ExternalMedia channel (`format=ulaw`, RTP) targeting it. Caller audio is
batched (100 ms) into `input_audio_buffer.append`; the model's
`response.output_audio.delta` (G.711 μ-law, no transcoding) is paced out as
20 ms RTP frames to Asterisk's `UNICASTRTP_LOCAL_ADDRESS/PORT` (or wherever
packets come from — symmetric RTP). Server VAD handles turn taking; when the
caller barges in, queued audio is flushed and the assistant item truncated.

Tools: `transfer_to_queue`, `end_call`, `create_callback`, `lookup_contact`,
`take_message`, `check_business_hours`. Handoff and hang-up wait until the
assistant has finished speaking. Transcripts (caller and AI) are stored and
streamed live to the panel; on handoff the AI summary is shown to the agent.

After every answered or AI call, the analyzer summarizes the transcript (or
transcribes the recording first) and stores summary, sentiment and a
suggested disposition. Voicemails are transcribed too.

## Persistence

SQLite (pure Go driver, WAL). Configuration objects are JSON documents in an
`entities` table; users, memberships, calls, transcripts, call events,
callbacks, voicemails, agent status events, settings and the audit log have
dedicated tables. Reports are SQL aggregates over `calls` plus a replay of
`agent_events` for time-in-status.

### Multi-tenancy

Every row and document carries a `tenant_id`. Store access requires a
context scope: `store.ForTenant(ctx, id)` for requests (the tenant comes
from the signed-in user) or `store.System(ctx)` for platform jobs; a context
without a scope fails closed (`ErrNoTenant`). Creating, updating or
deleting another tenant's object behaves as "not found". The call core
keeps per-tenant caches (flows by dialed number, queues, departments) and
tags every live call, agent and event with its tenant; the WebSocket and
webhooks only deliver a tenant's own events. Dialed numbers (`/numbers`)
decide which organization an inbound call belongs to.

Schema changes are versioned migrations (`schema_version`): v2
multi-tenancy, v3 Joow accounts, v4 tickets, v5 knowledge full-text index,
v6 imported emails, v7 routing decisions.

### Platform services

| Package | Responsibility |
|---|---|
| `internal/joow` | Joow identity, directory and notifications (HTTP or local provider) |
| `internal/tickets` | ticket workflow, SLA deadlines, history, notifications |
| `internal/secure` | AES-256-GCM with per-tenant keys derived (HMAC) from the master key |
| `internal/kb` | text extraction, chunking, FTS5 indexing, permission-aware search |
| `internal/mail` | IMAP connector: scoped fetch, parsing, threading, ticket / knowledge import |
| `internal/routing` | topic classification, rule scoring, target resolution with availability and escalation, rule drafting |
| `internal/retention` | hourly per-tenant purge of recordings, transcripts and email bodies |

Routing decisions for AI calls flow as: AI tool `route_call` → routing
engine (topics, approved rules, live agent state from the core, department
hours) → decision logged with steps → `core.AIRoute` hands the caller to the
chosen agent or queue with the AI's summary.

## Realtime & integrations

The event hub fans out `call.*`, `agent.*`, `callback.created`,
`voicemail.created` and `config.changed` to WebSocket clients (with a
snapshot on connect and every 10 s) and to signed webhooks
(`X-Callcenter-Signature: sha256=<hmac>`), retried 3× with backoff.
Prometheus metrics are exposed on `/metrics`.

## Security

- Bearer JWT (HS256). Authorization is permission-based: built-in roles
  admin / supervisor / agent map to permission sets, custom roles grant
  explicit permissions; the user, its role and its organization's status are
  re-checked on every request. Agents only see and control their own calls.
  Passwords use bcrypt. All changes go to the audit log; configuration keeps
  revision history.
- Organizations are isolated server-side (see Multi-tenancy). Knowledge
  files, email credentials and email bodies are encrypted at rest with
  per-tenant keys; recording playback/download, knowledge downloads and
  retention purges are audited.
- ARI and the media port range must not be exposed to the internet.
- The Electron app keeps the remote panel sandboxed with context isolation
  and grants only the microphone to the configured server origin.
