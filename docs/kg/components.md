# Components

Go module `callcenter-middleware` (Go 1.27, pure Go, no cgo), a React panel
(`web/`), an Electron app (`desktop/`), deployment files (`deploy/`) and a
real-PBX test harness (`test/asterisk-e2e/`).

## Dependency picture

```
cmd/middleware ── wires everything (main.go, services.go)
   │
   ├─ internal/server ── HTTP API, WebSocket, auth/permissions, SPA (embeds internal/web)
   │      uses: core, store, tickets, kb, mail, messages, routing, retention, joow, ai, events
   ├─ internal/core ── call core (single-goroutine actor) ── uses ari via domain.SwitchController
   │      ▲ AI sessions call back through core.AIControl
   ├─ internal/ai ── OpenAI Realtime bridge + tools ── uses core (types), routing, kb, tickets, store
   ├─ internal/routing ── routing engine ── uses store, core.IsOpen; live state via Live interface (core)
   ├─ internal/tickets ── ticket workflow ── uses store, joow; OnReply hook → mail, messages
   ├─ internal/kb ── knowledge files/search ── uses store, secure
   ├─ internal/mail ── IMAP/SMTP connector ── uses store, secure, tickets, kb
   ├─ internal/messages ── chat intake/replies ── uses store, tickets, joow
   ├─ internal/retention ── purge job ── uses store; deletes recordings via ari
   ├─ internal/joow ── Joow adapter (HTTP or local) ── RunAlerts consumes events
   ├─ internal/store ── SQLite persistence, tenant scoping, migrations
   ├─ internal/secure ── AES-GCM per-tenant encryption
   ├─ internal/events ── in-process pub/sub hub (WebSocket, webhooks, alerts)
   ├─ internal/notify ── signed webhooks + legacy screen-pop
   ├─ internal/ari ── Asterisk REST client + event stream
   ├─ internal/domain ── shared types and constants (no logic deps)
   └─ internal/config, metrics, seed, web
```

`domain` is the leaf everyone imports. `core` never imports `server`,
`ai`, `routing`, `tickets` etc.; collaborators are injected through
interfaces in `core/ports.go` (`AIManager`, `AIControl`, `Analyzer`) and
`domain` (`SwitchController`, `Publisher`, `MetricsRecorder`).

## Go packages

| Package | Owns | Key files | Notes / invariants |
|---|---|---|---|
| `cmd/middleware` | process wiring, background jobs | `main.go`, `services.go` | Order: store → Joow provider → bootstrap/seed → services → core → routing engine → HTTP. Background: webhooks, voicemail transcription, mail poller (1 min), retention (1 h), Joow call alerts. |
| `cmd/ari-sim` | simulated Asterisk ARI | `main.go` → `internal/sim` | Generates callers; `POST /sim/call` injects one. Default creds match config defaults. |
| `cmd/mail-sim` | local IMAP + SMTP sink | `main.go` | `POST :1080/deliver` adds mail; logs outgoing replies. |
| `internal/core` | live calls, agents, queues, IVR, ACD, transfers, consult/conference, monitoring, voicemail, callbacks, campaigns, survey, AI glue | `core.go` (actor loop), `inbound.go`, `ivr.go`, `acd*.go`, `answer.go`, `commands.go`, `consult*.go`, `finish.go`, `ai_glue.go`, `ai_control.go`, `cache*.go`, `departments.go` | **All state is mutated only on the actor goroutine** (`post`/`do`). A call = legs (channels) with roles (`types.go`). Per-tenant config cache reloaded on `config.changed`. Fail-open to Asterisk `app_queue` on errors. |
| `internal/ai` | OpenAI Realtime voice sessions over RTP, tools, post-call analysis, browser client secrets | `manager.go`, `realtime*.go`, `rtp*.go`, `session.go` (prompt), `tools.go` (definitions), `tool_calls.go` (dispatch), `directory*.go` (tool backends), `analyzer.go` | Session context is tenant-scoped. Every tool call is audited (`ai.<tool>`). |
| `internal/routing` | topic detection, rule scoring, target resolution, rule drafting | `engine.go`, `match.go`, `resolve.go`, `targets.go`, `text.go` (Persian normalization), `generate.go` | Only `approved` rules route. Resolution order: target → rule fallback → job-profile escalation contact → department queue. |
| `internal/server` | REST API (~120 routes + 16 CRUD resources), WebSocket, auth, permission checks, audit, SPA serving | `routes.go`, `authmw.go`, `crud.go`, `h_*.go`, `visibility.go`, `ratelimit.go` | Each request carries a principal (user, tenant, permissions) re-checked from the DB. Handlers return errors; `respond.go` maps them (404 for other tenants' objects). |
| `internal/store` | SQLite (modernc, WAL, 1 connection), repositories, migrations | `store.go`, `tenant.go`, `collection*.go`, `migrations*.go`, per-area files | Fail-closed tenant scope: every call needs `ForTenant`/`System` context. Generic JSON documents in `entities` via `Collection[T]`. |
| `internal/tickets` | ticket lifecycle, SLA, history, notifications | `service.go`, `update.go` | `OnReply` hook delivers public staff replies (email, Joow chat). |
| `internal/kb` | knowledge files (encrypted blobs), extraction, chunking, indexing, permission-aware search | `service.go`, `extract.go`, `pdf.go`, `search.go` | `Allowed()` is the single access rule for people and the AI. |
| `internal/mail` | IMAP fetch (read-only), parse, filters, ticket threading, KB import, SMTP replies, poller | `imap.go`, `parse.go`, `sync.go`, `ticket.go`, `manage.go`, `reply.go` | Dedup by `(connector, uid)`; threading by `[T-n]` or In-Reply-To/References. |
| `internal/messages` | chat → tickets, replies to Joow conversations | `service.go` | Conversation ↔ ticket mapping in `conversations`. |
| `internal/retention` | per-tenant purge policy and job | `retention.go` | Purges are audited (`retention.purge`). |
| `internal/joow` | Joow provider interface; `HTTP` (production) and `Local` (dev); call alerts | `joow.go`, `http.go`, `local.go`, `alerts.go` | Optional `Messenger` interface for chat replies. |
| `internal/secure` | AES-256-GCM, per-tenant keys via HMAC derivation | `secure.go` | `enc:` prefix for sealed strings. |
| `internal/domain` | entities, enums, permissions, events | `calls.go`, `config_entities.go`, `users.go`, `org.go`, `permissions.go`, `tickets.go`, `knowledge.go`, `email.go`, `routing.go`, `qa.go`, `conversation.go`, `events.go` | No imports of other internal packages. |
| `internal/ari` | ARI REST client + WebSocket event stream (with reconnect) | `client.go`, `ws_stream.go`, `originate.go`, `bridges.go`, `channels.go` | Implements `domain.SwitchController`. |
| `internal/events` | pub/sub hub | `hub.go` | Subscribers: WebSocket clients, webhooks, voicemail transcriber, Joow alerts. |
| `internal/notify` | signed webhook dispatcher, legacy screen-pop | | HMAC `X-Callcenter-Signature`. |
| `internal/seed` | bootstrap admin, demo data (two organizations) | `seed.go`, `demo*.go`, `joow.go` | `SEED_DEMO=true` only on empty DB parts. |
| `internal/sim` | ARI simulator implementation | | Used by integration tests and demos. |
| `internal/integration` | end-to-end tests through the real API against the simulator | `stack_test.go` (wiring), one file per area | The best executable documentation of API behaviour. |
| `internal/auth` | session tokens and password hashing | `token.go` | JWT HS256 with `JWT_SECRET`; bcrypt; the server re-checks the user and organization on every request. |
| `internal/config`, `metrics`, `web` | env config, Prometheus metrics, embedded SPA (`dist/`) | | `dist/` is committed and must match `web/` (CI). |

## Apps and other folders

| Path | What | Notes |
|---|---|---|
| `web/` | React 18 + Vite + TypeScript control panel | Pages under `src/pages/<area>`; API types `src/api/types*.ts`; permissions `src/lib/perms.ts`, `src/state/auth.tsx` (`useCan`); nav `src/components/layout/nav.ts`; routes `src/App.tsx`. Build output → `internal/web/dist`. |
| `desktop/` | Electron wrapper | Tray presence, native call notifications, sandboxed remote panel. See `desktop/README.md`. |
| `deploy/asterisk` | Asterisk 18 config (PJSIP, ARI, dialplan, queues, MOH) | `from-external` → `Stasis(callcenter-app)`. |
| `deploy/prometheus` | scrape config | |
| `test/asterisk-e2e` | real Asterisk in Docker + fake OpenAI Realtime (`fakert`) + scenario driver | `make e2e-asterisk`; ports 18080/18089/18099. |
| `docs/` | reference docs and this KG | |
