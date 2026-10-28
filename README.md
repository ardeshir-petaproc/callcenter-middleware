# Call Center Middleware

A complete call center platform built on **Asterisk ARI**, in a single Go
binary with an embedded web control panel, a desktop app and an **OpenAI
realtime voice agent** that callers can talk to.

- **Multi-tenant platform for Joow**: isolated organizations with
  departments (hours, capacity, priority, fallback), job profiles, employee
  profiles and custom roles with fine-grained permissions. Employees sign in
  with their Joow accounts (or by invitation); a platform admin manages
  organizations. See [docs/PRD_MAPPING.md](docs/PRD_MAPPING.md).
- **AI routing**: topics and routing rules (to an employee, a job profile, a
  queue or a department) that are drafts until approved; rules can be
  drafted by AI from job descriptions. Decisions use live availability and
  escalation paths and are logged with their reasons; a simulator shows
  where a request would go.
- **Tickets**: from calls, email, the AI or manually; SLA deadlines,
  internal notes vs replies, history, call links, reports.
- **Knowledge base**: files and notes encrypted at rest, scoped by
  department, role and category, with validity dates and Persian full-text
  search. The AI only uses approved, current, in-scope knowledge.
- **Email connector**: consented, folder-scoped IMAP import into tickets
  (with reply threading) or draft knowledge; revoke and purge at any time.
- **Quality and compliance**: scorecards and evaluations, audited
  recording access, recording notices, retention with automatic deletion.
- **AI voice agent**: callers talk to a GPT realtime agent, either AI-first
  or from an IVR option. It answers from a knowledge base, looks up the
  caller's CRM record, takes messages and schedules callbacks. When it needs
  a human it hands off to the right queue with a summary. You can also test a
  profile in the browser over WebRTC.
- **Native ACD**: queues with five strategies, penalties and skills, VIP
  priority, ring timeout, auto-pause on missed calls, wrap-up, SLA, max wait
  with overflow, leave-when-empty, callback-from-queue (`*`), recording, and
  fail-open to Asterisk `app_queue`.
- **Visual IVR flows** per dialed number: keypad menus, sub-menus, business
  hours and holidays, after-hours routing, direct (menu-less) entry points.
- **Agent desktop**:
  - Presence with pause reasons and a screen-pop showing the CRM contact, IVR
    path, AI summary and call history.
  - Call controls: hold, blind transfer to a queue, agent, number, AI or
    voicemail, warm transfer (consult) and conference, record, and hang up.
  - Wrap-up with dispositions, notes and tags.
  - Click-to-call, callbacks, and a WebRTC softphone.
- **Supervisor**: live wallboard, live calls with spy, whisper and barge,
  forcing agent status, live AI transcripts.
- **Admin**: users, queues and memberships, flows, schedules, AI profiles,
  dispositions, pause reasons, webhooks, settings, and an audit log.
- **Analytics**:
  - Call history with recordings, transcripts, AI summaries and sentiment.
  - Reports on SLA, abandon rate, ASA, AHT, agent time-in-status and
    hourly/daily volume, with CSV export.
  - Prometheus metrics.
- **Integrations**: signed webhooks for every realtime event, a REST API, a
  WebSocket event stream, and the legacy screen-pop webhook.
- **Desktop app** (Electron): tray presence switch, native incoming-call
  notifications, sandboxed panel.
- **ARI simulator**: demo and test everything without a PBX.

## Quick start (no PBX needed)

```bash
make web go                 # build the control panel + binaries (needs Node 22 and Go 1.27)
./bin/ari-sim -listen :8088 -rate 4 &          # simulated Asterisk with generated callers
./bin/mail-sim &                                # optional: local mailbox for the email connector
SEED_DEMO=true ./bin/middleware                 # http://localhost:8080
```

Log in as `admin` / `admin123`, or as a demo agent (`alice`, `bob`, `carol`,
`dave`; supervisor `sara`) with password `agent123`. A second organization,
`parsian` (`admin` / `admin123`, agent `reza` / `agent123`), shows tenant
isolation; "Sign in with Joow" works with the local Joow provider (demo
accounts `j-alice`, `j-bob`, …; `j-bob` belongs to both organizations). Set some agents to
**Available** and watch calls flow through the IVR into the queues. Inject a
scripted call:

```bash
curl -XPOST localhost:8088/sim/call -d '{"caller":"+15550100","dnis":"100","digits":"1"}'
```

To use the AI agent, set `OPENAI_API_KEY`. Then press 9 in the main menu, dial
DNIS `700`, or use **AI agents → Test in browser**.

## Production with Asterisk

```bash
cp .env.example .env        # set JWT_SECRET, ADMIN_PASSWORD, OPENAI_API_KEY, ARI credentials
docker compose up -d --build
```

The compose file runs the middleware, Asterisk 18 (configs in
`deploy/asterisk`) and Prometheus. Then:

1. Point your SIP trunk at the `trunk` endpoint in `deploy/asterisk/pjsip.conf`.
   Inbound calls run `Stasis(callcenter-app)` from `[from-external]`.
2. Register agent phones as `1001`–`1010`, or use the panel's softphone over
   `ws(s)://<asterisk>:8088/ws` (enable TLS in `http.conf` for `wss`).
3. In the panel, create users with matching extensions, queues and an IVR flow.
4. AI calls need Asterisk to reach the middleware's UDP media range
   (`MEDIA_ADVERTISE_HOST`, `MEDIA_PORT_MIN`–`MEDIA_PORT_MAX`).

Using an existing Asterisk: copy `ari.conf`, `http.conf` and the
`[from-external]` context, and keep an `app_queue` queue at `DEFAULT_QUEUE`
in `QUEUE_CONTEXT` for the fail-open path.

## Configuration

All settings are environment variables. `.env.example` lists every one with
its default. The most important:

| Variable | Default | Purpose |
|---|---|---|
| `HTTP_PORT` | `8080` | API, panel, metrics |
| `DB_PATH` | `data/callcenter.db` | SQLite database |
| `JWT_SECRET` | random per start | token signing secret (set it!) |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | `admin` / `admin123` | bootstrap admin (empty DB only) |
| `ARI_URL`, `ARI_WS_URL`, `ARI_USERNAME`, `ARI_PASSWORD`, `ARI_APP` | local Asterisk | ARI connection |
| `DEFAULT_QUEUE`, `QUEUE_CONTEXT` | `600`, `ext-queues` | fail-open target |
| `OPENAI_API_KEY` | — | enables AI agent, summaries and transcription |
| `OPENAI_REALTIME_MODEL`, `OPENAI_VOICE` | `gpt-realtime`, `marin` | voice agent defaults |
| `MEDIA_ADVERTISE_HOST`, `MEDIA_PORT_MIN/MAX` | `127.0.0.1`, `40000-40199` | RTP for AI calls |
| `OUTBOUND_DIAL_TEMPLATE`, `OUTBOUND_CALLER_ID` | `PJSIP/{number}@trunk` | click-to-call / callbacks |
| `AUTO_PAUSE_MISSES`, `DEFAULT_RING_TIMEOUT_SEC`, `DEFAULT_WRAPUP_SEC` | `3`, `20`, `10` | ACD defaults |
| `SEED_DEMO` | `false` | demo data |
| `DATA_DIR` | `data` | encrypted knowledge files and the generated key file |
| `DATA_ENCRYPTION_KEY` | generated into `DATA_DIR/.data-key` | master key for files and secrets at rest |
| `PUBLIC_URL` | `http://localhost:8080` | used in invitation links |
| `JOOW_MODE`, `JOOW_API_URL`, `JOOW_API_KEY` | `local` | Joow integration ([docs/JOOW_INTEGRATION.md](docs/JOOW_INTEGRATION.md)); use `http` in production |
| `JOOW_DEV_TOKENS` | `false` | local mode only: dev sign-in tokens (implied by `SEED_DEMO`) |
| `CORS_ORIGINS` | — | extra allowed origins |

When no IVR flow is configured, the original env-driven menu still works
unchanged (`GREETING_MEDIA`, `QUEUE_MAPPINGS`, `IVR_TIMEOUT_SEC`,
`MAX_RETRIES`, `WEBHOOK_URL`).

## Documentation

- **[docs/kg/README.md](docs/kg/README.md): start here** — the handover
  knowledge graph (status, components, flows, data model, conventions,
  operations, decisions, glossary). Agents: see [AGENTS.md](AGENTS.md).
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md): components, call lifecycle,
  fail-open behaviour, AI bridge, security.
- [docs/USER_GUIDE.md](docs/USER_GUIDE.md): roles and day-to-day workflows.
- [docs/API.md](docs/API.md): REST and WebSocket API with entity types.
- [docs/API_PLATFORM.md](docs/API_PLATFORM.md): organizations, permissions,
  tickets, knowledge, email, routing, QA and retention APIs.
- [docs/PRD_MAPPING.md](docs/PRD_MAPPING.md): PRD requirements → implementation,
  status, and proposals for the open technical decisions.
- [docs/JOOW_INTEGRATION.md](docs/JOOW_INTEGRATION.md): Joow identity,
  mini-app, notifications and call integration.
- [desktop/README.md](desktop/README.md): desktop app.

## Development

```bash
make test          # unit + integration tests (core, store, API, AI bridge, simulator stack)
make test-race
make check-loc     # Go files must stay <= 180 lines
make e2e-asterisk  # real Asterisk 18 in Docker: IVR, ACD, recording, hold, whisper,
                   # voicemail, click-to-call, CSAT survey, AI handoff over ExternalMedia, campaigns
cd web && npm run dev   # panel dev server on :5173, proxies /api to :8080
```

| Path | What |
|---|---|
| `cmd/middleware` | main binary |
| `cmd/ari-sim` | Asterisk ARI simulator |
| `cmd/mail-sim` | local IMAP/SMTP mailbox for the email connector |
| `internal/core` | call core: IVR, ACD, AI glue, voicemail, transfers, monitoring, outbound |
| `internal/ai` | OpenAI realtime bridge (RTP ⇄ WebSocket), tools, post-call analyzer |
| `internal/ari` | ARI REST client + event stream |
| `internal/server` | REST API, WebSocket hub endpoint, SPA serving |
| `internal/store` | SQLite persistence (tenant-scoped), migrations, reports |
| `internal/joow` | Joow identity / directory / notification adapter |
| `internal/tickets` | ticket workflow and SLA |
| `internal/kb`, `internal/secure` | knowledge base (extraction, search) and encryption at rest |
| `internal/mail` | IMAP email connector |
| `internal/routing` | routing engine, rule drafting from job profiles |
| `internal/retention` | retention policies |
| `internal/sim` | simulator implementation |
| `internal/integration` | end-to-end tests against the simulator |
| `test/asterisk-e2e` | end-to-end tests against real Asterisk (Docker) |
| `web/` | React control panel (built into `internal/web/dist`) |
| `desktop/` | Electron desktop app |
| `deploy/` | Asterisk and Prometheus configuration |
