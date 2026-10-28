# Joow integration

The call center runs as a service next to Joow. Employees keep their Joow
accounts; the control panel can be opened as a Joow mini-app; notifications
go to Joow. All Joow access goes through one interface,
`internal/joow.Provider`, with two implementations:

| `JOOW_MODE` | Provider | Use |
|---|---|---|
| `local` (default) | `joow.Local` — accounts in the `joow_accounts` table, HMAC dev tokens `jdev.<account>.<sig>` | development, demos, tests (`POST /api/v1/dev/joow-token {account_id}` mints a token when `SEED_DEMO=true` or `JOOW_DEV_TOKENS=true`) |
| `http` | `joow.HTTP` — the real Joow API at `JOOW_API_URL` with `JOOW_API_KEY` | production |

## What the middleware needs from Joow (HTTP contract)

```
GET  /v1/me                     Authorization: Bearer <Joow session token>  → Account
GET  /v1/accounts?q=&limit=     X-Api-Key: <service key>                     → {"accounts": [Account]}
GET  /v1/accounts/{id}          X-Api-Key                                    → Account
POST /v1/notifications          X-Api-Key  {"account_id", "kind", "title", "body", "url", "data"}
POST /v1/messages               X-Api-Key  {"conversation_id", "text"}   (reply to a customer conversation)
```

In the other direction, Joow forwards customer chat messages to the call
center: `POST /api/v1/inbound/messages?org=<slug>` with header
`X-Inbound-Key` (issued in the panel, `POST /integrations/inbound-key`) and
body `{channel: "joow", conversation_id, from_id, from_name, from_phone,
text}`. Conversations become tickets and staff replies go back to the
conversation.

`Account = {id, name, email, phone, avatar_url}`. Any equivalent Joow API
works by adapting `internal/joow/http.go` (≈100 lines); nothing else in the
code base talks to Joow directly.

Notification kinds sent today: `invitation`, `ticket_assigned` and
`incoming_call` (sent when an agent's phone starts ringing, with the caller
and a link to the agent console) and `routing_rule_pending` (to everyone
who may approve routing rules when drafts are submitted or generated).
Planned: `qa_evaluation`.

## Sign-in flows

1. **Inside Joow (mini-app).** Joow opens the panel URL and hands the
   mini-app its session token (URL fragment, `postMessage` or a JS bridge —
   whichever the Joow mini-app SDK provides). The panel calls
   `POST /api/v1/auth/joow {token}`. The middleware verifies the token with
   `GET /v1/me`, finds the employee records linked to that account
   (`users.joow_account_id`) and returns a session. If the person works for
   several organizations the response is `{choose_tenant: true, tenants}`
   and the panel asks which one (`tenant_id` on a second call). Switching
   later: `POST /auth/switch`.
2. **Invited staff.** An administrator invites by Joow account (search
   `GET /joow/accounts?q=`), email or phone. The invitee opens the link and
   accepts with their Joow token (the account is linked) or, if they have no
   Joow account yet, with a username and password.
3. **Direct add.** `POST /users/from-joow` creates an employee bound to a
   Joow account without an invitation.
4. **Break-glass.** Username/password login stays available for
   administrators (`POST /auth/login` with the organization slug).

Suspended organizations cannot sign in (platform admins excepted). Every
sign-in is audited (`login`, `login_joow`).

## Mini-app embedding checklist (Joow side)

- Allow the panel origin in the mini-app webview; the panel uses relative
  URLs (`/api/v1`, `/api/v1/ws`), so serving it under its own domain is
  simplest.
- Provide the session token to the webview (see above).
- Microphone permission in the webview if agents use the built-in WebRTC
  softphone.
- RTL layout: the panel renders user content with `dir="auto"`; a full
  Persian UI translation is a follow-up.

## Calls

The PRD asks to reuse Joow real-time calling. Today the media path is:

```
PSTN/SIP trunk ─┐                       ┌─ SIP desk phones
                ├─ Asterisk (ARI) ──────┼─ WebRTC softphone in the panel (JsSIP)
Joow (future) ──┘        ▲              └─ OpenAI Realtime (AI agent, RTP)
                         │ ARI
                 callcenter-middleware (routing, queues, AI, tickets)
```

Options for bringing Joow calls in, in order of preference:

1. **Joow ↔ SIP/WebRTC gateway.** Joow exposes calls to a business as SIP
   (or WebRTC with standard SDP) towards Asterisk. Customers calling a
   company inside Joow then enter the same IVR, AI and queues as phone
   callers; agents can answer in the panel softphone or — with the reverse
   direction — in their Joow app (Asterisk dials the agent's Joow identity
   through the gateway, endpoint configured per user as `endpoint`).
   No middleware change beyond configuring endpoints and a DNIS per company.
2. **Joow media server joins an Asterisk bridge** via ExternalMedia (the
   same mechanism the AI agent uses). Needs a Joow API to attach RTP to a
   call.
3. **Notification-only.** Joow is used for screen-pops and callbacks while
   media stays on SIP — possible today with `Notify`.

## Configuration

`JOOW_MODE` must be `local` or `http`; `http` without `JOOW_API_URL` and
`JOOW_API_KEY` refuses to start.

```
JOOW_MODE=http
JOOW_API_URL=https://api.joow.example
JOOW_API_KEY=...            # service key for directory and notifications
PUBLIC_URL=https://callcenter.example.com   # used in invitation links
```
