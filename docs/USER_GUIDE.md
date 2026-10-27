# User guide

## Roles

| Role | Can |
|---|---|
| **Agent** | Take calls, set their presence, wrap up calls, dial out, work callbacks and voicemail, edit CRM contacts, see their own history and stats. |
| **Supervisor** | Everything an agent can, plus: dashboards, the wallboard, all live calls and history, call control on any call, spy, whisper and barge, forcing an agent's status, reports, campaigns. |
| **Admin** | Everything, plus configuration (users, queues, flows, schedules, AI agents, dispositions, pause reasons, webhooks, settings) and the audit log. |

## Agents

1. Open the **Agent console** and set your status to **Available**. Calls are
   only offered to available agents. Use **Paused** with a reason for breaks;
   a pause you request during a call takes effect when the call ends.
2. When your phone rings, the console shows a screen-pop: the caller, their
   CRM record and VIP flag, the IVR path, the AI agent's summary (if the
   caller spoke to the AI first) and recent calls.
3. During the call you can **hold**, **transfer** (to a queue, a colleague,
   an outside number, the AI agent or voicemail), **record** and **hang up**.
4. After the call you are in **wrap-up** for the queue's wrap-up time. Pick a
   disposition and add notes and tags. You become available again
   automatically.
5. Missing several offered calls in a row (default 3) pauses you with the
   reason "Missed calls". Set yourself Available again when you are back.
6. **Click-to-call**: type a number in the dialer. Your phone rings first;
   when you answer, the customer is dialed. **Callbacks** work the same way
   from the callbacks list.
7. **Softphone**: in the console's softphone settings, enter your SIP
   WebSocket URL, SIP URI and password to take calls in the browser or the
   desktop app.

## Supervisors

- **Dashboard / wallboard**: live KPIs (waiting calls, longest wait, SLA,
  abandon rate, average wait and talk, CSAT), queue cards and the agent
  board. Queue cards turn red when a queue alert threshold is crossed.
- **Live calls**:
  - **Spy**: rings your extension and lets you listen.
  - **Whisper**: the agent hears you; the caller does not.
  - **Barge**: joins you into the conversation.
  - You can also transfer, hold or hang up any call, and read AI
    conversations live.
- **Reports**: volume by hour and day, per queue and per agent, time in each
  status, outcomes, dispositions, sentiment, CSAT, and CSV export.
- **Campaigns**: create a campaign for a queue, import leads (CSV
  `number,name,notes`) and press **Start**. Each available member of the
  queue is rung, then connected to the next lead. Unanswered leads are
  retried after the retry interval, up to the maximum attempts. Inbound
  callers waiting in that queue always come first.

## Admins

- **Users**: create users with a role and an extension (rung as
  `PJSIP/<extension>`, or a custom endpoint). Assign queues with a penalty;
  lower penalties are offered calls first. Skills must cover the queue's
  required skills.
- **Queues**:
  - Routing: strategy, ring timeout, max wait plus an overflow action, and
    wrap-up time.
  - Service levels: the SLA threshold, recording, callback-from-queue (the
    caller presses `*`), leave-when-empty, the post-call CSAT survey and
    alert thresholds.
  - Mode **asterisk** hands callers to an existing `app_queue` extension
    instead of the built-in ACD.
- **IVR flows**: bind a flow to a dialed number (`*` = default). Map keys to
  actions, and set the timeout action. Attach a schedule with an after-hours
  action, or use a *direct action* to skip the menu (e.g. an AI-first
  hotline).
- **AI agents**:
  - **Prompt and voice**: instructions, knowledge base, greeting, voice and
    language.
  - **Tools**: transfer, end call, callback, contact lookup, take message,
    business hours.
  - **Limits**: the handoff queue and a maximum duration.
  - **Test in browser** talks to the profile directly over WebRTC.
  - Requires `OPENAI_API_KEY`.
- **Schedules**: weekly opening hours per day plus holidays, in a time zone.
- **Settings**: dispositions, pause reasons, webhooks (HMAC-signed with
  `X-Callcenter-Signature`), company name, outbound recording, the audit log
  and system status.

## Media prompts

Prompts are Asterisk media URIs (for example `sound:custom/ivr-welcome`).
Upload your recordings to Asterisk's sounds directory (e.g.
`/var/lib/asterisk/sounds/custom/`). The defaults used by the middleware are
configurable through environment variables (`VOICEMAIL_MEDIA`,
`CALLBACK_MEDIA`, `CLOSED_MEDIA`, `SURVEY_MEDIA`, `SURVEY_THANKS_MEDIA`).
