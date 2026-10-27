# REST & Realtime API (v1)

Base path: `/api/v1`. All bodies are JSON. Timestamps are RFC 3339 UTC strings.
Errors: non-2xx status with `{"error": "message"}`.

Authentication: `POST /auth/login` returns a bearer token. Send it as
`Authorization: Bearer <token>`. For `<audio src>` and WebSocket URLs, pass it
as the `token` query parameter instead.

Roles: `admin` > `supervisor` > `agent`. "sup+" means supervisor or admin.

## Auth & self

| Method | Path | Role | Body / Query | Response |
|---|---|---|---|---|
| POST | `/auth/login` | public | `{username, password}` | `{token, expires_at, user}` |
| GET | `/auth/me` | any | | `{user, agent: AgentState}` |
| POST | `/auth/password` | any | `{current_password, new_password}` | `{ok:true}` |
| POST | `/me/status` | any | `{status: "available"\|"paused"\|"offline", reason?}` | `AgentState` |
| GET | `/me/stats` | any | | `Summary` (today, own calls) |
| POST | `/me/dial` | any | `{number}` | `{call_id}` (click-to-call: rings own phone then the number) |

## Live state

| Method | Path | Role | Response |
|---|---|---|---|
| GET | `/live` | any | `Snapshot` |
| GET | `/ws?token=` | any | WebSocket stream (see below) |

`Snapshot = {calls: CallView[], agents: AgentState[], queues: QueueView[], ai_sessions: number, alerts: QueueAlert[] /* currently active */, at}`

WebSocket messages are `{type, data, at}`:

- `snapshot` → `Snapshot` (sent on connect and every 10 s)
- `call.updated` → `CallView`
- `call.ended` → `CallView` or `Call` (final CDR, may arrive again after AI analysis)
- `agent.updated` → `AgentState`
- `call.transcript` → `TranscriptEntry`
- `agent.screenpop` → `{user_id, call: CallView}` (an agent's phone is ringing)
- `callback.created` → `Callback`
- `voicemail.created` → `Voicemail`
- `config.changed` → `{entity}`
- `queue.alert` → `{queue_id, name, level: "alert"|"clear", waiting, longest_wait_sec, available}` (thresholds `alert_waiting` / `alert_wait_sec` on the queue)

## Calls

| Method | Path | Role | Body / Query | Response |
|---|---|---|---|---|
| GET | `/calls` | any (agents: own) | `from,to` (RFC3339 or YYYY-MM-DD), `queue_id, agent_id, campaign_id, outcome, direction, q, limit, offset` | `{items: Call[], total}` |
| GET | `/calls/{id}` | any (agents: own) | | `{call: Call, live: CallView\|null, transcript: TranscriptEntry[], events: CallEvent[], contact: Contact\|null}` |
| PATCH | `/calls/{id}` | any (agents: own) | `{disposition?, notes?, tags?}` | `Call` |
| POST | `/calls/{id}/hangup` | sup+ or own agent | | `{ok}` |
| POST | `/calls/{id}/transfer` | sup+ or own agent | `{type: "queue"\|"agent"\|"external"\|"ai"\|"voicemail", target}` | `{ok}` |
| POST | `/calls/{id}/hold` | sup+ or own agent | `{on: boolean}` | `{ok}` |
| POST | `/calls/{id}/record` | sup+ or own agent | | `{ok}` |
| POST | `/calls/{id}/monitor` | sup+ | `{mode: "spy"\|"whisper"\|"barge"}` (rings the supervisor's own extension) | `{ok}` |
| GET | `/calls/{id}/recording` | any (agents: own) | `?token=` | `audio/wav` |

## Configuration (CRUD)

Each resource supports `GET /x` (list), `POST /x` (create), `GET /x/{id}`,
`PUT /x/{id}` (full replace), `DELETE /x/{id}`. Reads are allowed for every
role unless noted; writes require admin unless noted.

| Resource | Path | Entity |
|---|---|---|
| Users | `/users` | `User` + `password` on create/update (optional on update), `queues?: [{queue_id, penalty}]` |
| Queues | `/queues` | `Queue` |
| IVR flows | `/flows` | `IVRFlow` |
| Schedules | `/schedules` | `Schedule` |
| AI profiles | `/ai/profiles` | `AIProfile` |
| Dispositions | `/dispositions` | `Disposition` |
| Pause reasons | `/pause-reasons` | `PauseReason` |
| Webhooks | `/webhooks` (admin read) | `Webhook` |
| Contacts | `/contacts` (any role may write; `GET /contacts?q=`) | `Contact` |

Extra endpoints:

| Method | Path | Role | Body | Response |
|---|---|---|---|---|
| GET | `/members` | any | | `QueueMember[]` |
| PUT | `/queues/{id}/members` | admin | `[{user_id, penalty}]` | `QueueMember[]` |
| PUT | `/users/{id}/queues` | admin | `[{queue_id, penalty}]` | `QueueMember[]` |
| POST | `/users/{id}/status` | sup+ | `{status, reason?}` | `AgentState` |
| GET | `/contacts/lookup?phone=` | any | | `{contact: Contact\|null, recent_calls: Call[]}` |
| GET | `/ai/status` | any | | `{enabled, model, voice, voices: string[], tools: string[]}` |
| POST | `/ai/profiles/{id}/client-secret` | sup+ | | `{value, expires_at, model}` — ephemeral key to test the agent from a browser over WebRTC (`POST https://api.openai.com/v1/realtime/calls` with SDP) |

## Callbacks & voicemail

| Method | Path | Role | Body / Query | Response |
|---|---|---|---|---|
| GET | `/callbacks` | any | `status?` | `Callback[]` |
| POST | `/callbacks` | any | `{number, name?, queue_id?, notes?, due_at?}` | `Callback` |
| PUT | `/callbacks/{id}` | any | `Callback` fields | `Callback` |
| DELETE | `/callbacks/{id}` | sup+ | | `{ok}` |
| POST | `/callbacks/{id}/dial` | any | | `{call_id}` (rings the requesting user first) |
| GET | `/voicemails` | any | | `Voicemail[]` |
| PATCH | `/voicemails/{id}` | any | `{heard}` | `Voicemail` |
| DELETE | `/voicemails/{id}` | sup+ | | `{ok}` |
| GET | `/voicemails/{id}/audio` | any | `?token=` | `audio/wav` |

## Outbound campaigns

Progressive (agent-first) dialer: while a campaign is `running`, every
available member of its queue is rung and then connected to the next due
lead. Inbound callers waiting in that queue are always served first.

| Method | Path | Role | Body / Query | Response |
|---|---|---|---|---|
| GET/POST/PUT/DELETE | `/campaigns`, `/campaigns/{id}` | read any, write sup+ | `Campaign` | `Campaign` |
| POST | `/campaigns/{id}/start` / `/pause` | sup+ | | `Campaign` |
| GET | `/campaigns/{id}/leads` | any | `status?, limit, offset` | `Lead[]` |
| POST | `/campaigns/{id}/leads` | sup+ | `{leads: [{number, name?, notes?}]}` or `{csv: "number,name,notes\n..."}` | `{added, skipped}` |
| DELETE | `/campaigns/{id}/leads` | sup+ | `status?` | `{ok}` |
| GET | `/campaigns/{id}/stats` | any | | `{leads: Record<status|"total", number>, calls: Summary}` |

## Reports (sup+)

All accept `from, to, queue_id, agent_id`. Defaults: today.

| Path | Extra query | Response |
|---|---|---|
| `/reports/summary` | | `Summary` |
| `/reports/queues` | | `GroupStat[]` (`id` = queue id) |
| `/reports/agents` | | `GroupStat[]` (`id` = user id) |
| `/reports/agent-time` | | `AgentTime[]` (`seconds` per status) |
| `/reports/timeseries` | `by=hour\|day`, `tz_offset` (minutes east of UTC) | `Bucket[]` |
| `/reports/breakdown` | `by=outcome\|disposition\|sentiment` | `LabelCount[]` |
| `/reports/export.csv` | | CSV of calls |

## Admin & system

| Method | Path | Role | Response |
|---|---|---|---|
| GET | `/audit` | admin | `AuditEntry[]` (`limit, offset`) |
| GET | `/settings` | any | `map<string,string>` |
| PUT | `/settings` | admin | `map<string,string>` → merged map |
| GET | `/system/info` | any | `{version, ari_connected, ai_enabled, db_ok, uptime_sec, ws_clients, default_queue, queue_context, outbound_template, started_at}` |

Unauthenticated: `GET /healthz`, `GET /readyz`, `GET /metrics`.

## Entity shapes

```ts
type Role = "admin" | "supervisor" | "agent";
interface User { id: number; username: string; display_name: string; email: string; role: Role;
  extension: string; endpoint: string; skills: string[]; enabled: boolean; created_at: string; updated_at: string }
type AgentStatus = "offline" | "available" | "ringing" | "busy" | "wrapup" | "paused";
interface AgentState { user_id: number; username: string; display_name: string; extension: string;
  status: AgentStatus; pause_reason?: string; since: string; call_id?: string; last_call_end?: string;
  calls_today: number; talk_secs_today: number; missed_in_row: number; wrapup_until?: string; queue_ids: number[] }
type ActionType = "" | "queue" | "ai" | "voicemail" | "callback" | "hangup" | "flow" | "agent" | "external" | "dialplan";
interface Action { type: ActionType; target?: string; media?: string }   // target: queue id, AI profile id, flow id, user id, phone number or dialplan extension
interface Queue { id: number; name: string; description: string; mode: "native" | "asterisk"; extension: string;
  strategy: "ringall" | "longest_idle" | "round_robin" | "fewest_calls" | "random"; priority: number;
  ring_timeout_sec: number; max_wait_sec: number; wrapup_sec: number; sla_sec: number; moh_class: string;
  join_media: string; required_skills: string[]; record: boolean; allow_callback: boolean;
  leave_when_empty: boolean; survey: boolean /* post-call CSAT 1-5 by keypad */;
  alert_waiting: number /* 0 = off */; alert_wait_sec: number /* 0 = off */; overflow: Action; enabled: boolean }
interface Campaign { id: number; name: string; description: string; queue_id: number;
  status: "draft" | "running" | "paused" | "completed"; max_attempts: number; retry_minutes: number;
  schedule_id: number; script: string }
interface Lead { id: number; campaign_id: number; number: string; name: string;
  status: "pending" | "dialing" | "answered" | "failed" | "done"; attempts: number; next_attempt: string;
  call_id: string; notes: string; updated_at: string }
interface QueueMember { queue_id: number; user_id: number; penalty: number }
interface IVRFlow { id: number; name: string; dnis: string; greeting_media: string; invalid_media: string;
  timeout_sec: number; max_retries: number; options: Record<string, Action>; timeout_action: Action;
  direct_action: Action; schedule_id: number; after_hours_action: Action; enabled: boolean }
interface Schedule { id: number; name: string; timezone: string;
  weekly: { day: number /*0=Sun*/; start: string /*HH:MM*/; end: string }[]; holidays: { date: string; name: string }[] }
interface AIProfile { id: number; name: string; model: string; voice: string; language: string; instructions: string;
  greeting: string; temperature: number; tools: string[]; handoff_queue_id: number; max_duration_sec: number;
  knowledge_base: string; enabled: boolean }
interface Disposition { id: number; code: string; label: string; enabled: boolean }
interface PauseReason { id: number; label: string; enabled: boolean }
interface Webhook { id: number; url: string; events: string[]; secret: string; enabled: boolean }
interface Contact { id: number; name: string; phone: string; email: string; company: string; notes: string; tags: string[]; vip: boolean }
type CallState = "ivr" | "ai" | "queued" | "ringing" | "connected" | "hold" | "voicemail" | "dialing" | "message" | "survey" | "ended";
interface Call { id: string; direction: "inbound" | "outbound"; caller_number: string; caller_name: string; dnis: string;
  state: CallState; outcome: string; flow_id: number; queue_id: number; agent_id: number; ai_profile_id: number;
  contact_id: number; ivr_path: string; started_at: string; queued_at?: string; answered_at?: string; ended_at?: string;
  wait_sec: number; talk_sec: number; hold_sec: number; sla_met: boolean; disposition: string; notes: string;
  tags: string[]; recording: string; summary: string; sentiment: string; hangup_by: string;
  csat: number /* 0 = not rated, else 1-5 */; campaign_id: number }
interface CallView extends Call { ringing_agents: number[]; waiting_sec: number; duration_sec: number; monitors: number }
interface QueueView { id: number; name: string; waiting: number; longest_wait_sec: number; available: number;
  busy: number; paused: number; logged_in: number; members: number }
interface TranscriptEntry { call_id: string; speaker: "caller" | "ai" | "agent" | "call"; text: string; at: string }
interface CallEvent { call_id: string; type: string; detail: string; at: string }
interface Callback { id: number; call_id: string; number: string; name: string; queue_id: number;
  status: "pending" | "in_progress" | "done" | "failed" | "cancelled"; assigned_to: number; due_at?: string;
  attempts: number; notes: string; created_at: string; updated_at: string }
interface Voicemail { id: number; call_id: string; caller_number: string; caller_name: string; queue_id: number;
  recording: string; duration_sec: number; transcript: string; heard: boolean; created_at: string }
interface AuditEntry { id: number; user_id: number; username: string; action: string; entity: string; entity_id: string; detail: string; at: string }
interface Summary { total: number; inbound: number; outbound: number; queued: number; answered: number; abandoned: number;
  voicemail: number; callbacks: number; ai_resolved: number; sla_met: number; sla_percent: number; abandon_rate: number;
  avg_wait_sec: number; avg_talk_sec: number; max_wait_sec: number; total_talk_sec: number;
  csat_count: number; csat_avg: number }
interface GroupStat extends Summary { id: number }
interface AgentTime { user_id: number; seconds: Record<string, number> }
interface Bucket { key: string; total: number; answered: number; abandoned: number }
interface LabelCount { label: string; count: number }
```

Outcomes: `answered, abandoned, voicemail, callback_requested, ai_resolved,
sent_to_dialplan, failed, no_answer, ivr_hangup, transferred_external`.

Media references (`greeting_media`, `join_media`, ...) are Asterisk media
URIs such as `sound:custom/welcome`, `sound:hello-world` or
`recording:my-upload`.
