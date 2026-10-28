# Agent guide

Start with the knowledge graph: **[docs/kg/README.md](docs/kg/README.md)**
(status, components, flows, data model, conventions, operations, decisions,
glossary; machine-readable `docs/kg/graph.json`).

Non-negotiable rules (details in `docs/kg/conventions.md`):

- Every Go file stays ≤ 180 lines (`make check-loc`).
- All store access is tenant-scoped (`store.ForTenant` / `store.System`); never trust `tenant_id` from input.
- Guard routes with permissions (`requirePerm`, `anyPerm`, `can`), not roles.
- Live call state is mutated only inside the core actor (`post`/`do`); never call `do` from inside it.
- Audit sensitive actions; never return or log secrets; seal secrets at rest.
- The AI may only use approved rules and allowed, current knowledge.
- After changing `web/`, run `cd web && npm run build` and commit `internal/web/dist`.
- Keep docs in sync: API docs, `docs/kg/*.md`, and `docs/kg/graph.json`. `go test ./docs/kg` catches missing paths/links and packages without a node; prose accuracy is your job.

Verify before you finish:

```bash
go vet ./... && make check-loc && go test -race ./...
cd web && npm run typecheck && npm run build
make e2e-asterisk   # when call handling changed (needs Docker)
```
