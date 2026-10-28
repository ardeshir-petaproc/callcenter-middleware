# Glossary

| Term | Meaning here |
|---|---|
| **Joow** | The platform this call center is built for (messaging / super-app). Employees keep Joow accounts; the panel is meant to run as a Joow mini-app. |
| **Organization / tenant** | A company using the platform. Fully isolated data; identified by `tenant_id` and a `slug` used at sign-in. Default tenant id 1 (`default`). |
| **Platform admin** | Operator of the whole platform (bootstrap admin); manages organizations. |
| **Department** | Part of an organization with hours, capacity, priority and a fallback action. |
| **Job profile** | A job description: responsibilities, request types handled, authority limits, escalation conditions and contact. Basis for routing rules. |
| **Topic** | A kind of request (e.g. Refund) with keywords in any language; optional owning department. |
| **Routing rule** | Topic/keyword conditions → target (employee, job profile, queue, department) with fallback and priority. Draft → approved → rejected/retired. |
| **Routing decision** | Logged explanation of where a request was sent and why (steps, candidates). |
| **ARI** | Asterisk REST Interface + WebSocket events; how the middleware controls calls. |
| **Stasis** | Asterisk application state where ARI controls a channel (`Stasis(callcenter-app)`). |
| **Channel / leg** | One call party's connection in Asterisk. A call = caller leg + agent/consult/media/snoop legs. |
| **Bridge** | Asterisk mixing bridge joining legs (caller + agent, conferences). |
| **DNIS** | The dialed number; selects organization and IVR flow. |
| **IVR flow** | Keypad menu / direct entry point per dialed number. |
| **ACD** | Automatic call distribution (queues, strategies, penalties, skills). |
| **Strategy** | How agents are chosen: ringall, linear (sequential), longest_idle, round_robin, fewest_calls, random. |
| **Penalty** | Member priority tier in a queue (lower rings first). |
| **Wrap-up** | After-call time before an agent is available again. |
| **Blind transfer** | Move the caller without talking to the target first. |
| **Warm transfer / consult** | Agent talks to the target while the caller is on hold, then completes, conferences or cancels. |
| **Spy / whisper / barge** | Supervisor listens / talks to the agent only / joins the call (Asterisk snoop). |
| **ExternalMedia** | Asterisk channel that streams RTP audio to an external host — used for the AI agent. |
| **Realtime (OpenAI)** | Speech-to-speech model session (WebSocket) acting as the voice agent. |
| **AI profile** | Configuration of one voice agent: prompt, voice, tools, handoff queue, knowledge scope. |
| **Tool (AI)** | Function the voice agent can call (route_call, search_knowledge, create_ticket, …). |
| **Fail-open** | If the middleware cannot handle a call, Asterisk's own queue takes it so callers are never dropped. |
| **SLA** | Ticket targets per priority (first response, resolution); queue SLA = answer within N seconds. |
| **Internal note vs reply** | Ticket comments visible only to staff vs sent to the customer. |
| **Knowledge source** | A document, note or imported email in the knowledge base. |
| **Connector** | A consented mailbox connection (one folder) importing email into tickets or knowledge. |
| **Inbound key** | Per-organization secret allowing Joow (or another channel) to post chat messages. |
| **Scorecard / evaluation** | QA form with weighted (and critical) criteria / a reviewer's scoring of one call. |
| **Retention** | Automatic deletion of recordings, transcripts and email bodies after N days. |
| **Revision** | Stored snapshot of a configuration object after each change. |
| **KG** | This knowledge graph (`docs/kg`). |
