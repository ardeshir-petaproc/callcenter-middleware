# PRD v0.2 → implementation map

How the requirements of *"Intelligent call center and customer service
platform in Joow" (PRD v0.2)* map onto this repository, what is done, and
what remains. Everything lives in this repository for now (Go middleware +
React panel + Electron app); nothing needed splitting yet.

Legend: ✅ implemented and tested · 🟡 partly / needs a Joow-side counterpart · ⏳ not started

## 1. Product shape

| Requirement | Status | Where |
|---|---|---|
| Admin mini-app inside Joow | 🟡 | The control panel (`web/`) is a standalone SPA served by the middleware at `/`; it can be embedded as a Joow mini-app (webview) once Joow provides the mini-app shell and a token handoff — see [JOOW_INTEGRATION.md](JOOW_INTEGRATION.md) |
| Employees use their existing Joow accounts, no separate staff app | ✅ | `POST /auth/joow` (Joow token → session), `internal/joow` provider (HTTP contract + local dev provider), users linked by `joow_account_id` |
| Staff without a Joow account can be invited | ✅ | `/invitations` (link with expiring token), `/auth/accept-invite` (with Joow token or username/password) |
| Reuse Joow real-time calling | 🟡 | Telephony runs on Asterisk (SIP/PSTN + WebRTC softphone in the panel). Bridging Joow RTC calls needs a Joow media gateway — options in [JOOW_INTEGRATION.md](JOOW_INTEGRATION.md#calls) |

## 2. Organizations, structure, people

| Requirement | Status | Where |
|---|---|---|
| Strict multi-tenancy, enforced server-side | ✅ | `internal/store/tenant.go` (fail-closed tenant scope on every query), tenant-aware live state/events; tests `store/tenant_test.go`, `integration/tenant_test.go` |
| Company → department → team/queue → staff | ✅ | tenants, `/departments`, queues with `department_id`, queue membership |
| Employee in several departments | ✅ | `profile.department_ids` |
| Department hours, holidays, capacity, priority, fallback | ✅ | department `schedule_id` (schedules have holidays), `capacity`, `priority`, `fallback`; enforced in `core/departments.go` |
| Audit of configuration and role changes | ✅ | audit log on every change + full revision history (`/revisions`) |
| Employee profile (name, Joow id, title, department, team, manager, languages, hours, availability, skills, job description) | ✅ | `User.profile`, skills, live presence; job profiles |
| Job description: responsibilities, allowed request types, authority limits, escalation rules | ✅ | `/job-profiles` (+ bilingual templates) |
| AI turns job descriptions into structured rules, manager approval before activation | ✅ | `POST /routing/generate` → draft rules; `routing.approve` to activate; any edit returns to draft |
| History of changes | ✅ | job profile `version`, `/revisions` |
| Least-privilege RBAC | ✅ | permission model + custom roles (`domain/permissions.go`) |

## 3. Files, email and knowledge

| Requirement | Status | Where |
|---|---|---|
| Categories, tags, scope to department / profile / topic | ✅ | `KnowledgeSource` fields |
| Access levels | ✅ | `visibility` all/restricted + allowed roles + `kb.restricted` |
| Owner, last update, validity date (no stale answers) | ✅ | `owner_id`, `updated_at`, `version`, `valid_until`; expired sources are invisible to the AI |
| Explicit consent for email, never the whole mailbox by default | ✅ | consent required; one folder, start date, sender/subject filters; read-only select |
| Disconnect, revoke, remove from AI | ✅ | `DELETE /email-connectors/{id}?purge=1`, `POST /email-connectors/{id}/ai` |
| AI uses only tenant- and task-authorized sources | ✅ | `kb.Allowed`, AI profile scope (`kb_department_ids`, `kb_categories`), drafts excluded |
| Files encrypted at rest | ✅ | AES-256-GCM per-tenant keys (`internal/secure`); email credentials and bodies sealed |
| Full-text search incl. Persian | ✅ | SQLite FTS5 `unicode61` |
| Semantic (embedding) search | ⏳ | see decision D6 |
| OAuth mail (Gmail / Microsoft 365) | ⏳ | IMAP with app passwords today; see decision D7 |

## 4. Call center

| Requirement | Status | Where |
|---|---|---|
| Inbound / outbound via SIP / PSTN | ✅ | Asterisk ARI; trunks configured in Asterisk; click-to-call, campaigns |
| Blind and warm transfer, hold / resume | ✅ | `/calls/{id}/transfer`, `/consult` + complete/cancel, `/hold` |
| Return to previous operator when a transfer fails | ✅ | failed consult returns the caller; unanswered blind transfer to an agent rings the transferring agent again |
| Queues, priority, ring groups simultaneous or sequential | ✅ | strategies ringall, linear, longest_idle, round_robin, fewest_calls, random; penalties; VIP and department priority |
| IVR | ✅ | data-driven flows per number |
| Operator status | ✅ | available / paused (reasons) / wrap-up / offline |
| Conference | ✅ | `/consult/conference` |
| History, missed, waiting time | ✅ | call history, reports, live wallboard |
| Recording with controlled access | ✅ | `recordings.play` / `recordings.export`, audited, consent notice, retention |
| Supervisor tools | ✅ | live view, listen / whisper / barge, queue alerts |

## 5. AI voice assistant

| Requirement | Status | Where |
|---|---|---|
| Natural conversation (speech) | ✅ | OpenAI Realtime over RTP (`internal/ai`) |
| Routing from org structure, job descriptions, skills, knowledge, live availability | ✅ | `route_call` tool → `internal/routing` engine |
| Summary to the operator, escalation to a human | ✅ | handoff summary on the call, screen-pop |
| Create / update tickets | ✅ | `create_ticket`, `ticket_status`, `update_ticket` tools (caller-verified) |
| Log the routing reason; explainable, testable rules | ✅ | decision log with steps and candidates; `/routing/simulate` |
| Answers only from approved knowledge | ✅ | `search_knowledge` tool |

## 6. Ticketing

| Requirement | Status | Where |
|---|---|---|
| From call / message / email / manual | ✅ (message = Joow chat pending) | `source` call/email/manual/ai |
| Unique id, category, priority, status, owner, history | ✅ | `T-<n>` per tenant, history table |
| Internal notes vs customer-visible replies | ✅ | `internal` flag |
| Link to call, recording, AI summary | ✅ | `call_ids`, `/calls/{id}/ticket` |
| Statuses new → closed | ✅ | |
| Response / resolution / overdue reports | ✅ | SLA policies, `/tickets/stats`, `overdue=1` |
| Replies sent to the customer by email / Joow message | ✅ email / ⏳ Joow | SMTP per connector (`internal/mail/reply.go`); Joow messaging pending (D8) |

## 7. Quality and compliance

| Requirement | Status |
|---|---|
| Live view, recording search / play by permission | ✅ |
| Evaluations (scorecards, critical criteria, acknowledgement) | ✅ |
| Monitoring with permission | ✅ (`calls.monitor`) |
| Audit of play / download / policy changes | ✅ |
| Retention and automatic deletion | ✅ (recordings, transcripts, email bodies) |
| Consent | ✅ recording notice before agents ring; email consent |

## 8. Security

Tenant isolation server-side ✅ · RBAC least privilege ✅ · TLS in transit
(terminate at the reverse proxy; see deploy docs) ✅ · encryption at rest
for files, mail credentials and bodies ✅ (database file encryption is
decision D4) · audit of AI operations ✅ (routing decisions, AI tickets,
KB access) · revoke external sources ✅ · retention / export policies ✅
(retention purge, permission-gated recording export, audited organization
export `GET /export/organization.zip`).

## 9. Roadmap phases

| PRD phase | Delivered in |
|---|---|
| 0 Foundations | single binary, store, auth, ARI core, simulator, e2e harness |
| 1 Organization & identity | multi-tenancy, permissions, departments, job profiles, Joow adapter, invitations |
| 2 Call center core | ACD, IVR, transfers, consult/conference, recording, supervisor tools |
| 3 Ticketing | tickets, SLA, call links |
| 4 Knowledge & email | KB, encryption, FTS, IMAP connector |
| 5 AI routing | topics, rules, approval, engine, decision log, AI tools |
| 6 Quality & compliance | QA, retention, recording access control |
| 7 Joow mini-app & scale | 🟡 waiting on Joow APIs; Postgres / HA (D2, D3) |

## 10. Open technical decisions (§15) — proposals

| # | Decision | Proposal |
|---|---|---|
| D1 | How Joow calls reach the call center | Treat Joow RTC as another channel into Asterisk: a Joow ↔ SIP/WebRTC gateway (preferred) so Joow users dial a company and agents answer in Joow; until then agents use the panel's WebRTC softphone or SIP phones. Details in [JOOW_INTEGRATION.md](JOOW_INTEGRATION.md#calls). |
| D2 | Database | Keep SQLite (WAL) for single-node deployments; the store is behind repositories with tenant scoping, so moving to PostgreSQL (with row-level security as defence in depth) is mechanical when multi-node is needed. |
| D3 | Scale-out / HA | One middleware per Asterisk cluster today (call state is an in-memory actor). For HA: active/passive with shared DB and ARI reconnection (already reconciles live channels on reconnect). |
| D4 | Encryption key management | `DATA_ENCRYPTION_KEY` (or generated key file) with per-tenant derived keys now; move the master key to a KMS/HSM and add key rotation (re-seal on read) for production. Database-level encryption via encrypted volume. |
| D5 | Speech/AI provider and Persian quality | OpenAI Realtime now (good Persian). Abstracted behind `core.AIManager`, so a local STT/TTS (e.g. for data residency) can be added per tenant. |
| D6 | Knowledge retrieval | FTS5 keyword search now (works for Persian, cheap, explainable). Add embeddings (pgvector or SQLite vector extension) as a second ranking stage when the corpus grows. |
| D7 | Email access | IMAP + app password with folder/filter scoping now; add OAuth (Gmail, Microsoft Graph) with read-only scopes so consent and revocation happen at the provider too. |
| D8 | Customer replies channel | SMTP per connector for email tickets (done); Joow messages for customers on Joow once a messaging API exists. |
| D9 | Data residency & hosting | Deploy per region (single binary + Asterisk); tenants pinned to a deployment; recordings stay on the Asterisk host / object storage in-region. |
| D10 | Repository layout | Keep the monorepo-style layout (Go `cmd/` + `internal/`, `web/`, `desktop/`); split services only when an independent scaling or ownership need appears. |
