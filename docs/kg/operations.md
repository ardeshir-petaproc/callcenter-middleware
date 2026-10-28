# Operations runbook

## Run modes

| Mode | Command | Use |
|---|---|---|
| Local demo, no PBX | `make go && ./bin/ari-sim -rate 4 & SEED_DEMO=true ./bin/middleware` | development, demos, UI work |
| + email | `./bin/mail-sim` (IMAP `:1143` user `username`/`password`, security `none`; SMTP `:1025`; `curl -XPOST :1080/deliver -d 'from=a@b.com&subject=Hi&body=...'`) | email connector |
| Docker demo | `cp .env.example .env` (set `ARI_URL=http://172.28.0.30:8088/ari`, `ARI_WS_URL=ws://172.28.0.30:8088/ari/events`, `SEED_DEMO=true`) then `docker compose --profile demo up -d --build middleware ari-sim mail-sim` | demo on a server |
| Production | `cp .env.example .env`, set secrets, `docker compose up -d --build` (middleware + Asterisk 18 + Prometheus) | real calls |

Panel: `http://<host>:8080/`. Health: `/healthz` (process), `/readyz` (DB + ARI). Metrics: `/metrics`.

## Production checklist

- [ ] `JWT_SECRET` set (otherwise sessions reset on restart).
- [ ] `ADMIN_PASSWORD` changed (bootstrap admin is created only on an empty DB and is the platform admin).
- [ ] `DATA_ENCRYPTION_KEY` set from a secret store, **or** the generated `/data/.data-key` backed up. Without it, knowledge files and mail passwords cannot be decrypted.
- [ ] TLS in front of the middleware (reverse proxy); the panel's softphone needs `wss` to Asterisk (`http.conf` TLS).
- [ ] ARI (8088) and the RTP range (`MEDIA_PORT_MIN–MAX`, UDP) reachable only from Asterisk / the middleware, never from the internet.
- [ ] `MEDIA_ADVERTISE_HOST` = an address Asterisk can reach (AI calls).
- [ ] SIP trunk configured in `deploy/asterisk/pjsip.conf` (`trunk` endpoint); agent phones `1001–1010` or WebRTC.
- [ ] `PUBLIC_URL` = the public panel URL (invitation links, Joow notifications).
- [ ] Joow: `JOOW_MODE=http`, `JOOW_API_URL`, `JOOW_API_KEY` (see [../JOOW_INTEGRATION.md](../JOOW_INTEGRATION.md)). `JOOW_MODE=local` also enables `/dev/joow-token` — **never use local mode in production**.
- [ ] `OPENAI_API_KEY` for the AI agent, summaries and transcription.
- [ ] Egress rules: the server contacts OpenAI, Joow, and the IMAP/SMTP hosts admins configure.
- [ ] Retention policy set per organization (Settings → Retention & export).

All variables: `.env.example` (complete list) and the README table.

## Backups and restore

State = `DB_PATH` (with `-wal`/`-shm` files) + `DATA_DIR` (`kb/` blobs and `.data-key`) + Asterisk recordings spool. Consistent backup: stop the middleware (or snapshot the volume) and copy `/data` together with the key. Restore: put the files back and start the same version or newer; migrations run automatically on start. Keep the master key with (but separately protected from) the backups.

## Upgrades

Pull the new version, rebuild the image, restart. Versioned migrations (see [data-model.md](data-model.md)) run once at start; there are no down-migrations — back up first. On shutdown the process drains (`DRAIN_TIMEOUT_SEC`, `core.DrainAndRelease`): callers not yet talking to a person (IVR, AI, queue) are released to the default Asterisk queue; bridged conversations continue in Asterisk without the middleware. Restart during low traffic anyway.

## Multi-organization administration

The bootstrap admin is a **platform admin**: Platform page (`/platform`) creates organizations (slug + owner) and suspends/activates them (suspended orgs cannot sign in; inbound key disabled). Each organization's dialed numbers are registered under Phone numbers; a number belongs to exactly one organization and decides where calls go.

## Troubleshooting

| Symptom | Check |
|---|---|
| Calls go straight to the Asterisk queue / dialplan | Middleware not connected to ARI (`/readyz`, log "Connected to Asterisk ARI"); fail-open is working as designed. |
| A number rings the wrong organization or nobody | `/numbers` entry for that DNIS and its flow; the default organization handles unknown numbers. |
| AI calls are silent | `MEDIA_ADVERTISE_HOST` and UDP range reachable from Asterisk; `OPENAI_API_KEY`; logs "AI session". |
| Recording download 404 | Recording deleted by retention, or the Asterisk spool directory missing/not writable (`/var/spool/asterisk/recording`). |
| "credentials unavailable" on an email connector | Master key changed since the connector was created — restore the key or re-enter the password. |
| Email connector status `error` | `last_error` on the connector (login, folder name, TLS mode/port). Test with "Load folders". |
| Routing never picks a person | Rules must be **approved**; use the Routing → Simulator to see each step; agents must be Available. |
| AI ignores a document | It must be active, not expired, AI-enabled and within the AI profile's knowledge scope (Knowledge → AI preview). |
| Joow sign-in fails | `JOOW_MODE`; `GET /v1/me` reachable; the Joow account must be linked to an employee (`joow_account_id`) in that organization. |
| `docker compose` rejects `.env` | Values with spaces or `<>` must be quoted (see `OUTBOUND_CALLER_ID`). |
| Building the image behind a TLS-inspecting proxy | The build stages need the proxy CA; in the dev sandbox this was done with a temporary Dockerfile copy, never in the committed one. |
| `make e2e-asterisk` fails immediately | Docker daemon running; ports 18080/18089/18099 free; the `andrius/asterisk:18` image pullable. |
