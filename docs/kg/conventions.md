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
10. **Docs and KG stay in sync:** API changes → `docs/API.md` / `docs/API_PLATFORM.md`; new packages, flows, tables or decisions → the matching `docs/kg/*.md` page and `docs/kg/graph.json` (`go test ./docs/kg` checks every path in the graph exists).
11. Commit messages: imperative subject, a body explaining why; work on a feature branch.

## Recipes

### Add a configuration resource (CRUD document)
1. Type in `internal/domain` with `ID`, `TenantID` fields.
2. `Store`: field `X *Collection[domain.X]` and `newCollection(db, "x_kind", func(x *domain.X) (*int64, *int64) { return &x.ID, &x.TenantID })` in `store.Open`.
3. Routes: `mountCRUD(r, "/xs", s, s.d.Store.Xs, crudOpts[domain.X]{entity: "x_kind", writePerm: ..., history: true, prepare: validateX, check: s.checkXRefs})`. Use `s.exists(r, kind, id)` to verify references stay inside the tenant.
4. If the call core needs it, load it in `core/cache.go` and set `reload: true`.
5. Test in `internal/integration` (create, cross-tenant 404, permission 403).
6. Panel page under `web/src/pages/<area>`, types in `web/src/api/types_*.ts`, nav entry with a permission.

### Add a SQL table
New `store/migrations_vN.go`, append to `versioned`, repository functions using `scoped(ctx, "tenant_id")` for reads and `ownerFor(ctx, obj.TenantID)` for writes.

### Add an AI tool
1. Name constant in `domain` (`routing.go` holds the newer ones).
2. Definition in `ai/tools.go` (`add(...)` + `AllTools`).
3. Dispatch in `ai/tool_calls.go`; backend logic in `ai/directory_*.go` (tenant-scoped ctx is provided).
4. Prompt guidance in `ai/session.go: Instructions`.
5. Unit test in `ai/directory_org_test.go` style; enable it in the demo profile (`seed/demo.go`).

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
| KG links | `docs/kg/graph_test.go` | `go test ./docs/kg` |
