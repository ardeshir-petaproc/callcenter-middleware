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

## Organizations and people

- **Signing in.** Choose your organization (or use *Sign in with Joow*; if
  your Joow account works for several organizations you pick one and can
  switch later from the user menu).
- **Departments** have business hours, a capacity (concurrent calls), a
  priority and a fallback (e.g. voicemail) used when closed or full.
- **Job profiles** describe a role: responsibilities, request types it
  handles, authority limits and when to escalate (and to whom). Start from a
  template. Every save creates a new version.
- **Employees** are linked to Joow accounts. Add them from the Joow
  directory, or send an invitation link (they accept with Joow or create a
  password). Give them departments, a job profile, languages and a manager.
- **Custom roles** grant exactly the permissions you tick.

## Routing rules

Topics describe what callers ask for (with keywords in every language you
use). Rules send a topic or keywords to an employee, anyone holding a job
profile, a queue or a department, with an optional fallback. Rules start as
**drafts**; someone with the *approve routing* permission activates them,
and any later edit sends them back for approval. *Generate from job profile*
drafts rules from a job description. Use the **simulator** to see where a
request would go right now and why; the decision log shows the reason for
every AI-routed call.

## Tickets

Tickets come from calls (*Create ticket* on a call), email, the AI
assistant or are created by hand. Internal notes are never shown to the
customer; the first public reply counts as the first response for SLA.
Overdue tickets are flagged; SLA targets are set per priority.

## Knowledge and email

Upload documents or write notes, choose who may see them (everyone, or
restricted to roles), which departments they belong to, whether the AI may
use them and until when they are valid. Expired or draft knowledge is never
used by the AI. Email connectors import one folder of a mailbox you
explicitly consent to share, optionally only from certain senders; new
conversations become tickets and replies are added to them, or messages
become draft knowledge for review. You can withdraw the AI's access or
disconnect and purge a mailbox at any time.

## Transfers and conferences

*Consult* calls a colleague or a number while the caller hears music. Then
*Complete transfer* hands the caller over, *Conference* joins all three,
*Cancel* returns to the caller. If the colleague does not answer, the caller
comes back to you automatically; if you hang up after the colleague
answered, they keep the caller. A blind transfer to a colleague who does not
answer also comes back to you.

## Quality and retention

Reviewers score calls with scorecards (weighted criteria; a *critical*
criterion scored 0 fails the evaluation). Agents see final evaluations and
acknowledge them. Recording playback and downloads are permission-controlled
and logged. Retention settings delete recordings, transcripts and email
bodies after the number of days you choose.

## Media prompts

Prompts are Asterisk media URIs (for example `sound:custom/ivr-welcome`).
Upload your recordings to Asterisk's sounds directory (e.g.
`/var/lib/asterisk/sounds/custom/`). The defaults used by the middleware are
configurable through environment variables (`VOICEMAIL_MEDIA`,
`CALLBACK_MEDIA`, `CLOSED_MEDIA`, `SURVEY_MEDIA`, `SURVEY_THANKS_MEDIA`).
