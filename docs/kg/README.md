# Knowledge graph — start here

This folder is the handover knowledge base for **callcenter-middleware**:
everything a person or an agent needs to understand, run, change and extend
the system without any prior context. It is organized as a small graph:
topic pages (nodes, below) that link to each other and to code, plus a
machine-readable [`graph.json`](graph.json) whose links are checked by
`go test ./docs/kg` (so the KG cannot silently drift from the code).

## What this is (one paragraph)

A multi-tenant **intelligent call center and customer-service platform**
built for **Joow** (a messaging / super-app platform). One Go binary
controls **Asterisk** over ARI (queues, IVR, transfers, recording,
supervisor tools), runs an **OpenAI Realtime voice agent** that talks to
callers and routes them using the organization's approved routing rules,
and provides **tickets, an encrypted knowledge base, email and chat
intake, quality reviews and retention**. A React control panel is embedded
in the binary; an Electron desktop app wraps it. Every organization
(tenant) is isolated server-side; employees sign in with their Joow
accounts. The product requirements are a Persian PRD (v0.2), mapped
requirement by requirement in [../PRD_MAPPING.md](../PRD_MAPPING.md).

## Reading order

| # | Node | Read it to learn |
|---|---|---|
| 1 | [status.md](status.md) | what is done, what is not, known limitations, next steps |
| 2 | [components.md](components.md) | every package / app, what it owns, how they depend on each other |
| 3 | [flows.md](flows.md) | the important runtime flows end to end (calls, AI routing, tickets, email, chat, retention) |
| 4 | [data-model.md](data-model.md) | tables, document kinds, migrations, tenancy and encryption of data |
| 5 | [conventions.md](conventions.md) | rules of the code base and recipes for common changes |
| 6 | [operations.md](operations.md) | run locally, deploy, configure, back up, upgrade, troubleshoot |
| 7 | [decisions.md](decisions.md) | why things are the way they are (decision log) and open decisions |
| 8 | [glossary.md](glossary.md) | domain and telephony terms |

Reference documents (outside this folder):
[README](../../README.md) · [ARCHITECTURE](../ARCHITECTURE.md) ·
[API](../API.md) · [API_PLATFORM](../API_PLATFORM.md) ·
[USER_GUIDE](../USER_GUIDE.md) · [PRD_MAPPING](../PRD_MAPPING.md) ·
[JOOW_INTEGRATION](../JOOW_INTEGRATION.md) · [desktop/README](../../desktop/README.md)

## Five-minute start

```bash
make web go                                   # Node 22 + Go 1.27 (web build is committed, `make go` alone is enough)
./bin/ari-sim -listen :8088 -rate 4 &         # simulated Asterisk with generated callers
./bin/mail-sim &                              # optional: local IMAP/SMTP for the email connector
SEED_DEMO=true ./bin/middleware               # http://localhost:8080
```

(`make demo` does the same with `DB_PATH=data/demo.db`.)

Logins — the sign-in form's "Organization" field takes the slug:
- `default`: `admin`/`admin123`; supervisor `sara`/`agent123`; agents
  `alice`, `bob`, `carol`, `dave` / `agent123`.
- `parsian`: `admin`/`admin123`, agents `reza` and `bob` / `agent123`.
- "Sign in with Joow" uses the local dev provider (accounts `j-alice`,
  `j-bob`, …; `j-bob` works for both organizations, so it shows a picker).

Agents start **Offline**: set them Available (agent console or Live view)
before calls can ring. The simulator generates calls to numbers 100 (IVR)
and 700 (AI) of `default` only; for `parsian` (number 200) inject a call:
`curl -XPOST localhost:8088/sim/call -d '{"dnis":"200"}'` (or start the
simulator with `-dnis 100,200,700`).

## How to verify everything still works

```bash
go vet ./... && make check-loc && go test -race ./...   # unit + integration (simulator) tests, KG link check
cd web && npm run typecheck && npm run build             # panel; CI fails if internal/web/dist differs
make e2e-asterisk                                       # real Asterisk 18 in Docker (needs a Docker daemon)
```

Also part of CI: `cd desktop && npm run check`. `test/asterisk-e2e` is a
separate Go module (not covered by `./...` at the root) and is not run in
CI. `go test ./docs/kg` checks the KG's paths, links and coverage of
packages — not the correctness of prose; keep pages updated by hand.

Last full verification (2026-10-28, in the development sandbox): all of
the above passed, including the real-Asterisk scenarios queue, voicemail,
outbound, survey, AI, campaign and consult/conference; browser
click-through of every panel area with zero console/API errors; Docker
image build and boot. CI runs on `main`, `feat/**` and pull requests
(`.github/workflows/ci.yml`).

## Where the work happened

Branch `feat/callcenter-platform` of `ardeshir-petaproc/callcenter-middleware`
(no pull request opened yet). `git log --oneline` reads as a changelog;
each phase is a commit series: multi-tenancy & identity → tickets →
knowledge & email → routing → call handling, QA & retention → panel →
docs/KG.
