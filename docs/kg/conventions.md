# Conventions and recipes

## Rules (enforced or relied upon)

1. **Every Go file ≤ 180 lines** (`make check-loc`, CI). Split by concern (`h_x.go`, `h_x_save.go`, …). `web/` and `desktop/` are excluded but keep files small too.
2. **Tenant scope is mandatory.** Never call the store with an unscoped context; requests already carry `store.ForTenant`. Background jobs iterate tenants with `store.System` then scope each (`retention.Run`, `mail.pollDue`). Never trust `tenant_id` from a request body; handlers zero it (`src.TenantID = 0`) and the store sets it from the scope.
3. **Permissions, not roles,** guard routes: `requirePerm(domain.PermX)`, `anyPerm(...)`, `anyone`, `requirePlatform`; inside handlers use `can(r, perm)`. New capability → add a `Permission` in `domain/permissions.go` (and to `AllPermissions`, and `supervisorPerms` if supervisors get it by default) and to `web/src/lib/perms.ts`.
4. **Call state lives only in the core actor.** Mutate `liveCall`/`agentRT` only inside `c.post`/`c.do`/event handlers. Never call `c.do` from inside the actor (deadlock); APIs such as `Core.Agent` are for other goroutines.
5. **Audit sensitive actions** with `s.audit(r, action, entity, id, detail)`; configuration changes also get revisions (`crudOpts.history` or `s.revision`). Downloads, plays, exports, purges, consent, AI tool calls are audited.
6. **Secrets never leave the server:** strip them in responses (`publicKB`, `publicMailbox`, invitation `TokenHash`), seal them at rest (`secure.Box`), store only hashes for tokens/keys, remove them from exports (`secretKeys`).
7. **Errors:** handlers return errors; `bad(msg)` → 400, `errForbidden` → 403, `store.ErrNotFound` → 404, `core.ErrInvalidState` → 409 (`server/respond.go`).
8. **AI may only use approved, current, in-scope data:** rules must be `approved`; knowledge must pass `kb.Allowed` with `KBAccess{AI: true}`; ticket tools verify the caller owns the ticket.
9. **Embedded panel must be rebuilt** after changing `web/` (`cd web && npm run build`) and `internal/web/dist` committed — CI compares.
10. **Docs and KG stay in sync:** API changes → `docs/API.md` / `docs/API_PLATFORM.md`; new packages, flows, tables or decisions → the matching `docs/kg/*.md` page and `docs/kg/graph.json`. `go test ./docs/kg` only checks that paths and links exist and that every package and KG page has a node — prose must be kept accurate by hand.
11. Commit messages: imperative subject, a body explaining why; work on a feature branch.
12. Files near the 180-line limit (split them before adding code): `server/h_tickets.go`, `store/calls.go`, `server/h_org.go`, `integration/stack_test.go`, `domain/config_entities.go`. `make check-loc` lists offenders.

## Recipes

### Add a configuration resource (CRUD document)
1. Type in `internal/domain` with `ID`, `TenantID` fields.
2. `Store`: field `X *Collection[domain.X]` and `newCollection(db, "x_kind", func(x *domain.X) (*int64, *int64) { return &x.ID, &x.TenantID })` in `store.Open`.
3. Routes: inside the matching routes function in `internal/server` (`orgRoutes` in `h_org.go` for organization objects, `configRoutes` in `routes.go` for call-center config, or the area's own `xxxRoutes`) add `mountCRUD(r, "/xs", s, s.d.Store.Xs, crudOpts[domain.X]{entity: "x_kind", writePerm: ..., history: true, prepare: validateX, check: s.checkXRefs, present: hideSecrets})`. Use `s.exists(r, kind, id)` (extend its switch for new kinds) to verify references stay inside the tenant. `present` strips secrets from responses; seal secrets in `check` (see `h_webhooks.go`).
4. History: with `history: true`, also add the kind to `revisionPerms` (`server/h_org.go`) and, in the panel, to `RevisionEntity` (`web/src/api/types_org.ts`) and `REVISION_PERMS` (`web/src/components/RevisionsButton.tsx`).
5. If the call core needs it, load it in `core/cache.go` and set `reload: true`. Data that affects opening hours is read in several places: the core cache (`inbound.go`, `departments.go`, `campaigns.go`), the routing engine (`routing/engine.go: load`, `routing/targets.go`) and the AI's `Directory.BusinessHours` (`ai/directory.go`). Note `Schedule` already has holidays.
6. Test in `internal/integration` (create, cross-tenant 404, permission 403).
7. Panel: page under `web/src/pages/<area>`, types in `web/src/api/types_*.ts`, route in `web/src/App.tsx` (`Gate` with the permission), nav entry in `components/layout/nav.ts`; then `npm run build` and commit `internal/web/dist`.
8. Docs: API doc, the kinds table in `docs/kg/data-model.md`, and `graph.json` if it is a new feature.

### Add a SQL table
New `store/migrations_vN.go`, append to `versioned`, repository functions using `scoped(ctx, "tenant_id")` for reads and `ownerFor(ctx, obj.TenantID)` for writes.

### Add an AI tool
1. Name constant in `domain` (`routing.go` holds the newer ones).
2. Definition in `ai/tools.go` (`add(...)`) **and** `AllTools` — it is also the server's allow-list for AI profile tools (`server/h_ai.go`).
3. Dispatch in `ai/tool_calls.go: handleTool` (auditing as `ai.<tool>` is automatic); backend logic in `ai/directory_*.go` (tenant-scoped ctx is provided).
4. If the tool acts on the call itself (transfer, hang up, …): add a method to `core.AIControl` (`core/ports.go`), implement it in `core/ai_control.go` inside `c.post`, add it to the test recorder in `ai/realtime_test.go`, and call it via `s.afterPlayout` so the caller hears the AI finish first.
5. Prompt guidance in `ai/session.go: Instructions`.
6. Panel label in `web/src/pages/ai/AIProfileForm.tsx` (`TOOL_LABELS`), rebuild `dist`.
7. Unit test in `ai/directory_org_test.go` style; enable it in the demo profile (`seed/demo.go`); document it in `docs/API_PLATFORM.md`.

### Add a call-control feature
Core command (exported method on `*Core` using `withCall`) → leg role if a new channel is involved (`types.go`, `inbound.go: onLegAnswered`, `answer.go: onLegGone`, `finish.go` cleanup) → core test with the fake switch (`core/*_test.go`, `harness_test.go`) → API handler (`liveCallFor` checks ownership) → integration test via simulator → e2e scenario in `test/asterisk-e2e/driver` for real-Asterisk behaviour.

### Talk to Joow
Only through `joow.Provider` (and optional interfaces such as `joow.Messenger`). Add the HTTP call in `joow/http.go`, a logging/dev implementation in `joow/local.go`, and document the contract in `docs/JOOW_INTEGRATION.md`.

### Add a setting
Per-tenant key/values: `s.d.Store.GetSetting/SetSetting`. Keys editable through `PUT /settings` must be listed in `publicSettings` (`server/h_admin.go`) or start with `custom_`; internal keys (prefixes in `internalSettings`) are hidden from `GET /settings`. Process-level options are environment variables in `internal/config` — add them to `.env.example` and the README table.

## Testing map

| Layer | Where | Run |
|---|---|---|
| Unit | next to code (`*_test.go`) | `go test ./internal/<pkg>` |
| Core with fake switch | `internal/core/*_test.go` | `go test ./internal/core` |
| API end-to-end against the ARI simulator | `internal/integration` | `go test ./internal/integration` |
| Real Asterisk + fake OpenAI | `test/asterisk-e2e` | `make e2e-asterisk` (Docker) |
| Panel | TypeScript compiler; manual/Playwright click-through | `cd web && npm run typecheck && npm run build` |
| Desktop | TypeScript check | `cd desktop && npm run check` |
| KG links | `docs/kg/graph_test.go` | `go test ./docs/kg` |

CI (`.github/workflows/ci.yml`) runs Go vet/LOC/race tests, the web build
with the bundle check and the desktop check on `main`, `feat/**` and pull
requests. `test/asterisk-e2e` is a separate Go module (outside `./...`)
and only runs via `make e2e-asterisk`. `server`, `tickets`, `messages`,
`ari` and `seed` have no unit tests of their own; `internal/integration`
covers them.
