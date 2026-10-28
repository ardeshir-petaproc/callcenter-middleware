# Status

As of 2026-10-28, branch `feat/callcenter-platform`. Legend: ✅ done and
tested · 🟡 partial · ⏳ not started. Requirement-level detail:
[../PRD_MAPPING.md](../PRD_MAPPING.md).

## Done

| Area | Status | Evidence (tests) |
|---|---|---|
| Call core: IVR, native ACD (6 strategies incl. sequential), skills, penalties, VIP, overflow, wrap-up, auto-pause, voicemail, callbacks, click-to-call, campaigns, CSAT survey | ✅ | `internal/core/*_test.go`, `internal/integration`, `make e2e-asterisk` |
| Transfers: blind, warm (consult), conference, return to previous agent, hold, recording notice | ✅ | `internal/core/consult_test.go`, `integration/callcenter_test.go`, e2e `consult` |
| Supervisor: live view, spy / whisper / barge, queue alerts, wallboard | ✅ | e2e `queue` |
| AI voice agent (OpenAI Realtime over RTP) with tools: route_call, search_knowledge, create/lookup/update ticket, callbacks, messages, contact lookup, hours, transfer | ✅ | `internal/ai/*_test.go`, e2e `ai` (fake realtime server) |
| Multi-tenancy (server-side isolation), permissions, custom roles, platform admin | ✅ | `store/tenant_test.go`, `integration/tenant_test.go`, `permissions_test.go` |
| Organization: departments (hours, capacity, priority, fallback), job profiles, numbers, employee profiles, revision history | ✅ | `core/department_test.go`, integration tests |
| Joow identity (token sign-in, multi-org picker, switch), invitations, directory lookup, notifications (invite, ticket assigned, incoming call, rules pending) | ✅ with local provider; 🟡 real Joow API is a proposed contract | `joow/*_test.go`, `integration/tenant_test.go` |
| Tickets: SLA, notes vs replies, history, call links, stats, from call / email / chat / AI / manual | ✅ | `integration/tickets_test.go`, `messages_test.go`, `email_test.go` |
| Knowledge base: encrypted files, extraction (docx/pptx/xlsx/pdf/html/md/csv/txt), FTS5 (Persian), scope & validity, AI preview | ✅ | `kb/kb_test.go`, `integration/kb_test.go` |
| Email connector: consented, folder-scoped IMAP → tickets (threaded) or draft knowledge; SMTP replies; revoke/purge | ✅ | `mail/parse_test.go`, `integration/email_test.go`, browser run against `mail-sim` |
| Chat intake (`/inbound/messages`) → tickets, replies back via Joow | ✅ (Joow side pending) | `integration/messages_test.go` |
| Routing engine: topics, rules with approval, AI-drafted rules, availability/escalation, decision log, simulator | ✅ | `routing/engine_test.go`, `integration/routing_test.go`, `ai/directory_org_test.go` |
| Quality: scorecards, evaluations, acknowledgement, summary | ✅ | `integration/callcenter_test.go` |
| Compliance: audited recording/KB access, AI action audit, retention purge, organization export | ✅ | `retention/retention_test.go`, `integration/export_test.go` |
| Control panel for all of the above, desktop app, Docker image, CI | ✅ | `npm run typecheck/build`, browser click-through, `.github/workflows/ci.yml` |

## Not done / limitations (honest list)

| # | Item | Why / what is needed |
|---|---|---|
| L1 | Joow mini-app embedding | Needs Joow's mini-app shell to pass a session token (panel already accepts `?joow_token=`). See [../JOOW_INTEGRATION.md](../JOOW_INTEGRATION.md). |
| L2 | Joow voice calls into the call center | Needs a Joow ↔ SIP/WebRTC gateway (decision D1). Agents use SIP phones or the panel's WebRTC softphone today. |
| L3 | Real Joow API | `joow.HTTP` implements a *proposed* contract (`/v1/me`, `/v1/accounts`, `/v1/notifications`, `/v1/messages`); adapt `internal/joow/http.go` when Joow publishes theirs. |
| L4 | Persian (fa) UI translation | Panel UI strings are English; user content renders RTL correctly (`dir="auto"`). Add an i18n layer to `web/`. |
| L5 | Single node | Live call state is in memory (one middleware per Asterisk); SQLite. Scale-out plan: decisions D2/D3. |
| L6 | Key management | One master key (`DATA_ENCRYPTION_KEY` or generated `DATA_DIR/.data-key`) with per-tenant derived keys; no rotation or KMS yet (D4). Losing the key loses KB files and mail credentials. |
| L7 | Knowledge retrieval | Keyword FTS only (no embeddings, D6); PDFs need a text layer (no OCR); email attachments are listed by name, not imported. |
| L8 | Email | IMAP/SMTP with passwords (no OAuth for Gmail/M365, D7); replies are plain text without attachments. |
| L9 | Routing classification | Topic = AI-provided topic or keyword match; no learned classifier. One approver suffices (no two-person rule). |
| L10 | Retention scope | Purges recordings, transcripts and email bodies; call records, tickets and audit are kept. |
| L11 | Not performed | Load/performance testing; formal security review / pen test; accessibility audit. |
| L12 | Admin-supplied hosts | Email connector hosts are contacted from the server (admins with `settings.manage` could probe internal hosts). Restrict egress in production. |

## Suggested next steps (priority order)

1. Agree the Joow API contract with the Joow team (L3) → mini-app token handoff (L1) → RTC gateway (L2).
2. Persian UI translation (L4).
3. KMS-backed master key + rotation (L6); deploy behind TLS with egress rules (L12).
4. Load test (concurrent calls per Asterisk, AI sessions per node) and a security review (L11).
5. PostgreSQL + HA plan when more than one node is needed (L5).
6. Embeddings for knowledge retrieval and OAuth mail (L7, L8).
7. Open a pull request from `feat/callcenter-platform` once the above is triaged.
