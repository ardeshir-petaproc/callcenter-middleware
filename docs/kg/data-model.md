# Data model

One SQLite database (`DB_PATH`, WAL mode, pure-Go driver `modernc.org/sqlite`,
a single connection) plus encrypted files under `DATA_DIR`. Timestamps in
SQL tables are Unix **milliseconds** (`store.ms`), except the `emails` table
(Unix seconds).

## Tenancy rule

Every row and document has `tenant_id` (default tenant id **1**,
`core.DefaultTenant`). Store calls require a scoped context:
`store.ForTenant(ctx, id)` (requests, per-tenant jobs) or
`store.System(ctx)` (platform jobs that read across tenants). Unscoped
calls fail with `store.ErrNoTenant`. Reading or writing another tenant's
object returns `store.ErrNotFound` (HTTP 404). Global tables without a
tenant: `tenants`, `joow_accounts`, `schema_version`, legacy `settings`.

## Configuration documents (`entities` table)

JSON documents stored by `store.Collection[T]` with columns
`(id, kind, data, tenant_id, created_at, updated_at)`. Kinds and Go types:

| kind | type (`internal/domain`) | API |
|---|---|---|
| `queue` | `Queue` (strategy, penalties via `queue_members`, skills, record, record_notice, survey, overflow, department) | `/queues` |
| `flow` | `IVRFlow` (DNIS, menu options → `Action`) | `/flows` |
| `schedule` | `Schedule` (weekly ranges, holidays, timezone) | `/schedules` |
| `ai_profile` | `AIProfile` (model, voice, prompt, tools, handoff queue, KB scope) | `/ai/profiles` |
| `disposition`, `pause_reason` | wrap-up codes, pause reasons | `/dispositions`, `/pause-reasons` |
| `webhook` | `Webhook` (url, secret, events) | `/webhooks` |
| `contact` | `Contact` (CRM, VIP) | `/contacts` |
| `campaign` | `Campaign` (leads in `leads` table) | `/campaigns` |
| `department` | `Department` | `/departments` |
| `job_profile` | `JobProfile` (versioned) | `/job-profiles` |
| `phone_number` | `PhoneNumber` (DNIS → tenant + flow; unique across tenants) | `/numbers` |
| `custom_role` | `CustomRole` (permission list) | `/roles` |
| `invitation` | `Invitation` (token hash, expiry) | `/invitations` |
| `sla_policy` | `SLAPolicy` per priority | `/sla-policies` |
| `kb_source` | `KnowledgeSource` (metadata; blob on disk) | `/kb/sources` |
| `email_connector` | `EmailConnector` (sealed password) | `/email-connectors` |
| `topic`, `routing_rule` | `Topic`, `RoutingRule` | `/routing/topics`, `/routing/rules` |
| `qa_scorecard`, `qa_evaluation` | `QAScorecard`, `QAEvaluation` | `/qa/...` |

## Relational tables

| Table | Purpose |
|---|---|
| `tenants` | organizations (slug, name, status active/suspended, plan) |
| `users` | employees; `UNIQUE(tenant_id, username)`; role, custom_role_id, joow_account_id, platform_admin, status, `profile` JSON (title, departments, manager, languages, schedule, job profile, phone) |
| `queue_members` | queue ↔ user with penalty; row order = sequential ring order |
| `calls` | CDRs (id = call UUID; state, outcome, agent, queue, timings, recording name, AI summary/sentiment, csat, campaign) |
| `call_events`, `transcripts` | call timeline, speech transcript lines |
| `callbacks`, `voicemails`, `leads` | work items |
| `agent_events` | presence history (time-in-status reports) |
| `audit_log` | every change and sensitive access (action, entity, user, detail) |
| `revisions` | full JSON snapshot per configuration change |
| `tenant_settings` | per-tenant key/values (company name, retention.*, inbound.key_hash, recordings.agent_playback, …) |
| `joow_accounts` | accounts of the local (development) Joow provider |
| `tickets`, `ticket_comments`, `ticket_events` | tickets (number `T-<seq>` per tenant), replies/notes, history |
| `kb_fts` | FTS5 index (`unicode61 remove_diacritics 2`) of knowledge chunks |
| `emails` | imported messages (sealed body, Message-ID, references, linked ticket / KB source); `UNIQUE(connector_id, uid)` |
| `routing_decisions` | decision log (JSON) per call / rule |
| `conversations` | chat thread (channel, external id) ↔ ticket |
| `schema_version` | current migration version |

## Migrations

`store.migrate` runs the idempotent base schema (`migrations.go`, plus
additive column migrations that tolerate "duplicate column"), then
versioned steps once each (`migrations_v2.go: versioned`):

| v | Name | Change |
|---|---|---|
| 1 | base | original single-tenant schema |
| 2 | `multiTenantV2` | tenants, `tenant_id` everywhere, users rebuilt with per-tenant usernames, revisions, tenant settings |
| 3 | `joowV3` | `joow_accounts` |
| 4 | `ticketsV4` | tickets, comments, events |
| 5 | `knowledgeV5` | `kb_fts` |
| 6 | `emailV6` (`migrations_v6.go`) | `emails` |
| 7 | `routingV7` | `routing_decisions` |
| 8 | `conversationsV8` | `conversations` |

Add a migration: new `migrations_vN.go` with a `var xxxVN = []string{...}`
and append `{N, xxxVN}` to `versioned`. Never edit a released step.
`store/tenant_test.go` covers upgrading a single-tenant database.

## Files and encryption

| Data | Where | Protection |
|---|---|---|
| Knowledge originals | `DATA_DIR/kb/<tenant>/<uuid>.bin` | AES-256-GCM, key = HMAC(master, tenant) (`internal/secure`) |
| Email passwords, email bodies | DB (`enc:` sealed strings) | same |
| Master key | `DATA_ENCRYPTION_KEY` or generated `DATA_DIR/.data-key` (0600) | **back it up with the database** |
| Passwords | `users.password_hash` | bcrypt |
| Invitation tokens, inbound keys | DB | SHA-256 hashes only |
| Call recordings | Asterisk spool (`/var/spool/asterisk/recording`), streamed through the API | permission-checked, audited, retention-deleted |
