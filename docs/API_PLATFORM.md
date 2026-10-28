# Platform API (organizations, people, tickets, knowledge, routing, QA)

This extends [API.md](API.md). Base path `/api/v1`, JSON bodies, bearer
token auth, errors as `{"error": "..."}`.

## Organizations and permissions

Every organization (tenant) is fully isolated: every request runs inside the
caller's organization and objects of other organizations behave as if they
did not exist (404). Access is governed by **permissions**. Built-in roles
grant defaults (`admin` = all, `supervisor` = operations, `agent` = own
work); a **custom role** grants an explicit list. `GET /permissions`
returns `{all: string[], mine: string[]}`; the login response carries
`permissions` too.

| Permission | Grants |
|---|---|
| `org.manage` | departments, job profiles, revision history |
| `users.manage` | employees, invitations, Joow lookups, custom roles |
| `queues.manage`, `flows.manage`, `ai.manage`, `campaigns.manage`, `settings.manage` | configuration areas (settings also covers email connectors and retention) |
| `routing.manage` | topics, draft routing rules, rule generation, simulation |
| `routing.approve` | approve / reject / retire routing rules (activates them) |
| `kb.manage` / `kb.restricted` | manage knowledge / read restricted knowledge |
| `tickets.all` | every ticket (otherwise own + own departments') |
| `calls.all`, `calls.control`, `calls.monitor` | all calls / control any live call / listen-whisper-barge |
| `recordings.play` / `recordings.export` | play any recording / download recordings |
| `qa.evaluate` | scorecards and evaluations |
| `reports.view`, `audit.view` | reports / audit log |

### Sign-in

| Method | Path | Body | Response |
|---|---|---|---|
| POST | `/auth/login` | `{username, password, tenant?: slug}` (default `default`) | `{token, expires_at, user, tenant, permissions}` |
| POST | `/auth/joow` | `{token: <Joow token>, tenant_id?}` | session as above, or `{choose_tenant: true, tenants: [{id, slug, name, user_id, role}], account}` when the Joow account works for several organizations |
| GET | `/auth/tenants` | | organizations the signed-in Joow account belongs to |
| POST | `/auth/switch` | `{tenant_id}` | new session in that organization |
| GET | `/auth/invite/{token}` | public | `{organization, name, email, role, expires_at, joow_account_id}` |
| POST | `/auth/accept-invite` | `{token, joow_token?}` or `{token, username, password, display_name}` | session |
| POST | `/dev/joow-token` | `{account_id}` (local Joow mode only) | `{token}` — simulates Joow sign-in |

### Organization structure (`org.manage` to write; any employee may read)

| Resource | Path | Entity |
|---|---|---|
| Departments | `/departments` | `{name, description, manager_id, schedule_id, capacity, priority, fallback: Action, languages[], enabled}` |
| Job profiles | `/job-profiles` | `{title, department_id, description, responsibilities[], request_types[], authority_limits, escalation_conditions, escalate_to_user_id, is_template, version, updated_at}` (version increments on every save) |
| Phone numbers | `/numbers` (`flows.manage`) | `{number, label, flow_id, enabled}` — unique across all organizations |
| Custom roles | `/roles` (`users.manage`) | `{name, description, permissions[]}` |

`GET /job-profile-templates` returns ready-made job descriptions (English
and Persian). `GET /revisions?entity=<kind>&entity_id=<id>` returns the
change history `[{action, data (JSON snapshot), changed_by_name, at}]` for
departments, job profiles, numbers, roles, users, topics, routing rules,
scorecards.

### Employees

`User` gains `joow_account_id`, `custom_role_id`, `status`
(`active|invited|disabled`) and `profile: {title, department_ids[],
manager_id, languages[], schedule_id, job_profile_id, phone}`.

| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/joow/accounts?q=` | | `[{id, name, email, phone}]` — Joow directory search |
| POST | `/users/from-joow` | `{joow_account_id, role, custom_role_id?, profile?, extension?}` | `User` (no password: signs in with Joow) |
| GET | `/invitations` | | `Invitation[]` |
| POST | `/invitations` | `{name, email?, phone?, joow_account_id?, role, custom_role_id?, department_ids[]}` | `{invitation, link, token}` |
| POST | `/invitations/{id}/revoke` | | `Invitation` |

### Platform administration (platform admins only)

| Method | Path | Body |
|---|---|---|
| GET | `/platform/tenants` | — |
| POST | `/platform/tenants` | `{slug, name, plan, owner: {username, password, display_name, email, joow_account_id}}` |
| PUT | `/platform/tenants/{id}` | `{name, plan, status: "active"\|"suspended"}` |

## Tickets

| Method | Path | Body / Query | Response |
|---|---|---|---|
| GET | `/tickets` | `status, priority, category, department_id, assignee_id, mine=1, overdue=1, call_id, q, limit, offset` | `{items: Ticket[], total}` |
| POST | `/tickets` | `{subject, description, category, priority: low\|normal\|high\|urgent, department_id, assignee_id, contact_id, requester_name, requester_phone, requester_email, tags[]}` | `Ticket` |
| GET | `/tickets/{id}` | | `{ticket, comments, history, calls, overdue}` |
| PATCH | `/tickets/{id}` | any of `subject, description, category, priority, status, department_id, assignee_id, contact_id, tags` | `Ticket` |
| POST | `/tickets/{id}/comments` | `{body, internal}` (internal = staff-only note) | `TicketComment` |
| POST | `/tickets/{id}/link-call` | `{call_id}` | `Ticket` |
| POST | `/calls/{id}/ticket` | optional overrides | `Ticket` prefilled from the call (summary, transcript excerpt) |
| GET | `/tickets/stats` | `from, to` (`reports.view`) | counts, response/resolution averages, overdue |
| CRUD | `/sla-policies` | `{priority, first_response_minutes, resolution_minutes}` | |

`Ticket = {id, number: "T-1001", subject, description, category, priority,
status: new|assigned|in_progress|waiting_customer|resolved|closed, source:
call|email|manual|ai, department_id, assignee_id, contact_id,
requester_*, call_ids[], tags[], created_by, created_at, updated_at,
first_response_at, resolved_at, closed_at, response_due_at,
resolution_due_at}`. Changes publish `ticket.updated` on the WebSocket.

## Knowledge base

| Method | Path | Body / Query | Response |
|---|---|---|---|
| GET | `/kb/sources` | `q, category` | `KnowledgeSource[]` the caller may read |
| GET | `/kb/sources/{id}` | | `{source, text}` (extracted text, first 20k chars) |
| GET | `/kb/sources/{id}/download` | `?token=` | original file (decrypted; audited `kb.download`) |
| POST | `/kb/sources` | JSON `{...source, content: "text"}` or multipart `meta` (JSON) + `file` | `KnowledgeSource` (`kb.manage`) |
| PUT | `/kb/sources/{id}` | same; omit content/file to keep it | `KnowledgeSource` |
| DELETE | `/kb/sources/{id}` | | |
| GET | `/kb/search` | `q, limit` | `[{source_id, title, category, snippet, score}]` |
| POST | `/kb/search/ai` | `{q, profile_id?}` | what the AI would retrieve (with passage `text`) |
| GET | `/kb/categories` | | `[{category, count}]` |

`KnowledgeSource = {id, title, kind: file|text|email, category, tags[],
department_ids[], queue_ids[], job_profile_ids[], topics[], visibility:
all|restricted, allowed_roles[] ("admin"|"supervisor"|"agent"|"role:<id>"),
ai_enabled, status: active|draft|archived, valid_until: YYYY-MM-DD,
owner_id, version, updated_at, file_name, mime_type, size, checksum,
text_chars, connector_id, email_id}`. Files: txt, md, csv, json, html,
docx, pptx, xlsx, pdf (text layer), max 25 MB. The AI only uses sources
that are active, not expired, AI-enabled and within its profile's
`kb_department_ids` / `kb_categories`.

## Email connectors (`settings.manage`)

| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/email-connectors` | | `EmailConnector[]` (no passwords; `has_password`) |
| POST | `/email-connectors` | `{name, host, port, security: tls\|starttls\|none, username, password, folder, from_filter[], subject_filter, since_date, mode: ticket\|kb, department_id, ticket_category, kb_category, ai_enabled, poll_minutes, enabled, consent: true}` | `EmailConnector` (consent is mandatory) |
| POST | `/email-connectors/folders` | connection settings (+ `id` to reuse the stored password) | `string[]` folder names |
| PUT | `/email-connectors/{id}` | same (empty password keeps the stored one) | |
| POST | `/email-connectors/{id}/sync` | | `{imported}` |
| POST | `/email-connectors/{id}/ai` | `{enabled}` | `{sources_changed}` — allow/withdraw AI use of its knowledge |
| GET | `/email-connectors/{id}/messages` | `limit` | imported messages (no bodies) with linked `ticket_id` / `kb_source_id` |
| DELETE | `/email-connectors/{id}?purge=1` | | disconnect (credentials deleted); `purge=1` also deletes imported mail and knowledge |

Status fields: `status: never|ok|error, last_sync, last_error, imported,
consent_by, consent_at`.

Optional outgoing mail: `smtp_host, smtp_port, smtp_security
(tls|starttls|none), from_address`. When set, public staff replies on
tickets that came in through the connector are emailed to the customer
(subject `Re: [T-1001] …`, threaded with In-Reply-To); the customer's
answer threads back onto the ticket. A failed delivery adds an internal
note to the ticket.

## Routing

| Method | Path | Permission | Body | Response |
|---|---|---|---|---|
| CRUD | `/routing/topics` | routing.manage | `{name, description, keywords[], department_id}` | |
| GET | `/routing/rules?status=` | manage or approve | | `RoutingRule[]` |
| POST / PUT | `/routing/rules[/{id}]` | routing.manage | `RoutingRule` | saved as **draft** (an edit of an approved rule needs re-approval) |
| DELETE | `/routing/rules/{id}` | routing.manage | | |
| POST | `/routing/rules/{id}/approve` · `/reject` · `/retire` | routing.approve | `{note?}` | `RoutingRule` |
| POST | `/routing/generate` | routing.manage | `{job_profile_id}` | `{method: ai\|template, rules: RoutingRule[]}` (drafts) |
| POST | `/routing/simulate` | routing.manage | `{text, topic?, language?, vip?}` | `RoutingDecision` |
| GET | `/routing/decisions` | routing.manage or reports.view | `call_id, rule_id, limit` | `RoutingDecision[]` |

`RoutingRule = {id, name, topics[], keywords[], languages[], vip_only,
target_type: user|job_profile|queue|department, target_id, fallback: Action,
priority, rationale, status: draft|approved|rejected|retired, source:
manual|ai, job_profile_id, version, created_by, approved_by, approved_at,
updated_at}`.

`RoutingDecision = {call_id, at, request, topic, rule_id, action: Action,
target_name, reason, steps: string[], candidates: [{rule_id, rule_name,
score, matched[], result: selected|unavailable|skipped}]}`.

AI profile tools: `route_call`, `search_knowledge`, `create_ticket`,
`ticket_status` and `update_ticket` (in addition to the earlier ones). The
ticket tools only work on tickets linked to the caller's number or CRM
contact.

## Call control additions

| Method | Path | Body |
|---|---|---|
| POST | `/calls/{id}/consult` | `{type: "agent"\|"external", target}` — caller is held, the target rings |
| POST | `/calls/{id}/consult/complete` | hand the caller to the consulted party |
| POST | `/calls/{id}/consult/conference` | three-way call |
| POST | `/calls/{id}/consult/cancel` | drop the consulted party, resume the caller |
| GET | `/calls/{id}/recording?download=1` | download (`recordings.export`); playing needs `recordings.play` or own call |

`CallView` gains `consult: ringing|talking|conference` and
`consult_agent_id`. Queues gain strategy `linear` (sequential in member
order) and `record_notice` (media played before agents are rung).

## Quality assurance

| Method | Path | Permission | Body |
|---|---|---|---|
| CRUD | `/qa/scorecards` | qa.evaluate | `{name, description, pass_score, criteria: [{key, label, weight, max_score, critical}], enabled}` |
| GET | `/qa/evaluations?agent_id=&call_id=` | any (agents: own final) | |
| GET | `/qa/evaluations/{id}` | | `{evaluation, scorecard}` |
| POST / PUT | `/qa/evaluations[/{id}]` | qa.evaluate | `{call_id, scorecard_id, scores: {key: n}, notes: {key: text}, comment, status: draft\|final}` |
| POST | `/qa/evaluations/{id}/acknowledge` | the evaluated agent | `{comment}` |
| GET | `/qa/summary` | qa.evaluate or reports.view | `{agents: [{agent_id, evaluations, avg_score, passed, auto_failed}]}` |

Evaluation results: `score` (weighted %), `auto_failed` (a critical
criterion scored 0), `passed` (≥ `pass_score`).

## Retention (`settings.manage`)

| Method | Path | Body |
|---|---|---|
| GET / PUT | `/retention` | `{recording_days, transcript_days, email_days}` (0 = keep) |
| POST | `/retention/run` | apply now → `{recordings, transcripts, emails, failed}` |

`GET /export/organization.zip` (`settings.manage` + `audit.view`) downloads
the organization's data: configuration documents (secrets removed), users,
calls, tickets with comments and history, and the audit log. Audited as
`organization.export`.

The policy also runs hourly. Purges are audited (`retention.purge`).
Setting `recordings.agent_playback` (`"1"` default, `"0"` to disable) controls
whether agents may play their own calls' recordings.
