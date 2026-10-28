# Decision log

Decisions already taken (with the reason) and decisions still open. Open
technical decisions from the PRD (§15) are D1–D10 in
[../PRD_MAPPING.md](../PRD_MAPPING.md#10-open-technical-decisions-15--proposals);
they are summarized at the end.

## Taken

| ID | Decision | Why | Consequence / revisit when |
|---|---|---|---|
| A1 | Single Go binary controlling Asterisk over **ARI** (Stasis app), not AMI/AGI or a custom SIP stack | Full call control (bridges, originate, snoop, ExternalMedia) with a mature PBX; one deployable | One middleware per Asterisk; see D3 for HA |
| A2 | **Single-goroutine actor** for live call state (`core.post/do`) | No locks around complex call state; deterministic tests with a fake switch | Never block in the actor; long work goes to goroutines |
| A3 | **Native ACD** in the middleware, `app_queue` only as fail-open | Skills, penalties, VIP, departments, AI handoff, sequential ring need app-level control | If the middleware is down, Asterisk queue `DEFAULT_QUEUE` still takes calls |
| A4 | AI voice via **OpenAI Realtime** bridged with **ExternalMedia RTP** (G.711 µ-law) | Low latency, good Persian, no transcoding | Needs UDP reachability between Asterisk and middleware |
| A5 | **SQLite (WAL, pure Go)**; config objects as JSON documents (`Collection[T]`) | Zero-ops single binary; fast iteration on config shapes | Single writer; move to Postgres for multi-node (D2) |
| A6 | Multi-tenancy by **`tenant_id` + fail-closed context scope** in the store, not a DB per tenant | Strong isolation enforced in one place; easy cross-tenant platform jobs | All new queries must use `scoped`/`ownerFor`; tested in `store/tenant_test.go` |
| A7 | **Permission-based RBAC** with built-in role defaults and custom roles | PRD asks for least privilege and flexible roles | UI and API both check permissions |
| A8 | **Joow behind one interface** (`joow.Provider`) with a local dev provider | Joow APIs were not available; everything is testable locally | Real contract may differ → only `joow/http.go` changes |
| A9 | Routing rules are **drafts until approved**; any edit returns to draft; AI only drafts | PRD: AI-generated rules need manager approval; explainability | Approvers are notified in Joow |
| A10 | Routing engine is **deterministic** (keyword topics + scored rules + availability) with the LLM only choosing the topic | Explainable, testable (`/routing/simulate`), cheap | Add a learned classifier later if keyword topics are insufficient |
| A11 | Knowledge: **FTS5 keyword search**, AES-GCM encrypted originals, validity dates, draft status for imported email | Persian works with unicode61; explainable; no extra infra | Embeddings later (D6) |
| A12 | Email: **IMAP read-only select of one folder** + explicit consent; replies via SMTP of the same connector; threading by `[T-n]` | PRD: never the whole mailbox; revocable | OAuth providers later (D7) |
| A13 | Chat intake via a **generic inbound endpoint** with per-organization key | Joow (or any channel) can push messages without new code | Joow must call it; replies need `POST /v1/messages` on Joow |
| A14 | Warm transfer **keeps the original agent bridged** while the caller is on hold | Natural "return to caller" on failure; conference is just un-holding the caller | Consult leg is a separate role (`consult`) |
| A15 | **Every Go file ≤ 180 lines** | Keeps units reviewable by people and agents | Split files by concern |
| A16 | Panel built with Vite and **committed** to `internal/web/dist` (embedded) | `go build` alone produces a complete server | CI checks the bundle matches the source |
| A17 | Docs-as-code **KG** (`docs/kg`) with a link-checking test | Handover without prior context; prevents drift | Update KG with every structural change |

## Open (proposals in PRD_MAPPING §10)

| ID | Topic | Proposal in one line |
|---|---|---|
| D1 | Joow calls → call center | Joow ↔ SIP/WebRTC gateway into Asterisk |
| D2 | Database | Stay on SQLite single-node; Postgres (+RLS) when multi-node |
| D3 | HA / scale-out | Active/passive with shared DB; ARI reconnect already reconciles |
| D4 | Key management | KMS/HSM master key + rotation |
| D5 | Speech provider / residency | OpenAI now; local STT/TTS per tenant via `AIManager` |
| D6 | Knowledge retrieval | Add embeddings as a second ranking stage |
| D7 | Email access | OAuth (Gmail, Microsoft Graph) read-only scopes |
| D8 | Customer reply channels | SMTP (done) + Joow `POST /v1/messages` (contract proposed) |
| D9 | Data residency / hosting | Per-region deployments, tenants pinned |
| D10 | Repository layout | Keep this monorepo layout until a split is needed |
