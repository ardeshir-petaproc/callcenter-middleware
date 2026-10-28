# Runtime flows

Each flow lists the path through the code. Function names are stable
anchors (search for them).

## F1. Inbound call → IVR → queue → agent

1. Asterisk dialplan `[from-external]` runs `Stasis(callcenter-app)`; the ARI stream (`ari/ws_stream.go`) emits `EventCallStarted`; the dialed number (DNIS) is the channel's dialplan extension (`ari/normalize.go`).
2. `core.Run` → `handleEvent` → `onStasisStart` → `startInbound` (`inbound.go`): new `liveCall`; **tenant resolved from the dialed number** (`cache.tenantFor`: exact digits-only match against enabled `/numbers`; unknown/disabled → tenant 1); contact lookup (VIP → priority); flow chosen by `cache.flowFor` (the number's flow, else a flow with that DNIS, else the catch-all flow; a tenant without flows goes to its first queue).
3. Business hours: outside the flow's schedule the after-hours action runs. Otherwise the flow's direct action or the IVR menu (`ivr.go`, DTMF). Actions are executed by `actions.go: execute` (`queue`, `agent`, `ai`, `voicemail`, `callback`, `external`, `flow`, `dialplan`, `hangup`); a queue in `asterisk` mode is handed to the Asterisk dialplan.
4. `acd.go: enqueue` → `departmentGate` (closed / over capacity → department fallback) → optional **recording notice** (`announcing`, agents not rung until it ends) → join media + MOH → `dispatch`.
5. `acd_select.go: selectAgents` picks agents by strategy (ringall, linear, longest_idle, round_robin, fewest_calls, random; penalties; skills). `ring` originates the agent leg.
6. `answer.go: onAgentAnswered` bridges caller + agent, starts recording if the queue records, marks the agent busy.
7. Hang-up → `finish.go: finish` writes the CDR, agent → wrap-up; post-call AI analysis (summary, sentiment) runs asynchronously.

Every state change publishes `call.updated` (tenant-tagged) → WebSocket clients of that tenant and webhooks.

## F2. AI voice agent and AI routing

1. Action `ai` → `ai_glue.go: startAI` creates an ExternalMedia channel; `ai.Manager.Start` opens an RTP port and an OpenAI Realtime WebSocket (tenant-scoped context).
2. `session.go: BuildSession` composes instructions + tools for the AI profile (tools enabled per profile).
3. When the caller needs a person the model calls **`route_call`** (`tool_calls.go`) → `Directory.Route` → `routing.Engine.Decide`:
   topic (AI-given or keyword match) → only **approved** rules scored (priority, then score) → target resolved with live agent state and department hours → rule fallback → job-profile escalation contact → department queue → if no rule applied, the topic's department → decision **saved** (`routing_decisions`, also `ai_route` in the call timeline). A decision that errors is not logged.
4. After the assistant finishes speaking (`afterPlayout`), `core.AIRoute` stops AI and executes the decided action (agent, queue, or a fallback such as voicemail / outside number) with the AI summary on the call (screen-pop shows it). No decision → the AI profile's handoff queue. `/routing/simulate` runs the same `Decide` without logging (drafts cannot be simulated).
5. Other tools: `search_knowledge` (only active, unexpired, AI-enabled, in-scope sources), `create_ticket`, `ticket_status` / `update_ticket` (only the caller's own tickets), callbacks, messages, contact lookup, hours, `transfer_to_queue`, `end_call`. Every tool call is written to the audit log.

## F3. Warm transfer / conference

`POST /calls/{id}/consult` → `core.Consult` holds the caller (MOH), originates a `consult` leg (agent or external). On answer it joins the agent in the call bridge (`onConsultAnswered`).
- `consult/complete` → original agent released (wrap-up), consult leg promoted to agent/external, caller resumed.
- `consult/conference` → caller resumed into the bridge (three-way).
- `consult/cancel` or consult leg fails → caller back to the original agent.
- Original agent hangs up mid-consult → answered consult takes over; otherwise caller re-queued.
- Blind transfer to an agent who does not answer → `returnToPrevious` rings the transferring agent.

## F4. Ticket lifecycle

`tickets.Service.Create` (SLA deadlines from `sla_policies`) → `Update` (history in `ticket_events`, stamps resolved/closed, reopen) → `Comment` (internal note or public reply; first public staff reply = first response). Public staff replies trigger `OnReply` → `mail.Reply` (email tickets) and `messages.Reply` (Joow chat tickets). Assignment notifies the assignee in Joow. Visibility: `tickets.all`, else own + own departments.

## F5. Email intake

Connector created with explicit consent (`POST /email-connectors`, password sealed). Poller (`mail.Service.Run`, every minute; per-connector interval) or `POST /sync` → `IMAP.Fetch` (read-only select of one folder, UID > last, since date) → `Parse` (plain text preferred, HTML → text, quoted history stripped, legacy charsets) → `Matches` (sender/subject filters) → `SaveEmail` (sealed body, dedup) →
- mode `ticket`: `[T-n]` in subject or In-Reply-To/References → comment on that ticket **only if the sender is the ticket's requester and it is not closed** (a resolved/waiting ticket goes back to in progress); otherwise a new ticket;
- mode `kb`: draft knowledge source (needs review before the AI can use it).
A failed import is forgotten and the folder position is not advanced, so the next sync retries it. Disconnect deletes credentials; `?purge=1` also deletes imported mail and knowledge.

## F6. Chat intake (Joow conversations)

Joow posts to `/inbound/messages?org=<slug>` with `X-Inbound-Key` (hash stored in tenant settings) → `messages.Service.Receive`: existing conversation → customer reply on its ticket (reopen if resolved/waiting); new or closed → new ticket + `conversations` row. Staff replies go back via `joow.Messenger.SendMessage`.

## F7. Sign-in and tenancy

- Password: `POST /auth/login {username, password, tenant}` (rate limited).
- Joow: `POST /auth/joow {token}` → `Provider.VerifyToken` → employees with that `joow_account_id` → session, or organization picker; `/auth/switch` changes organization.
- Every authenticated request: `authenticate` loads the user (must be enabled, organization active), computes permissions (`domain.PermissionsFor`, custom role), and puts `store.ForTenant(ctx, tenant)` into the request context.

## F8. Knowledge upload and search

`POST /kb/sources` (multipart or JSON text) → `kb.Service.Save`: extract text, seal original into `DATA_DIR/kb/<tenant>/<uuid>.bin`, store metadata (`kb_source` document), re-index chunks in FTS5 (`kb_fts`). Search: FTS5 bm25 → filter by `kb.Allowed` (status, validity, AI flag, scope, visibility, roles, departments).

## F9. Retention and export

Hourly `retention.Service.Run` per tenant: recordings older than N days deleted on Asterisk (`DELETE /recordings/stored/{name}`) and references cleared; transcripts and email bodies purged; audited. `GET /export/organization.zip` streams configuration (secrets removed), users, calls, tickets and audit.
