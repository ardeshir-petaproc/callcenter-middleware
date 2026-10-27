import{c as s,j as n,az as l,I as o,f as t,V as c,B as d,K as i,ay as r}from"./index-DpvNHzdC.js";import{C as v}from"./clock-CwGTLedV.js";import{P as x,C as u}from"./phone-call-rc_QXJRW.js";/**
 * @license lucide-react v0.460.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const y=s("Hourglass",[["path",{d:"M5 22h14",key:"ehvnwv"}],["path",{d:"M5 2h14",key:"pdyrp9"}],["path",{d:"M17 22v-4.172a2 2 0 0 0-.586-1.414L12 12l-4.414 4.414A2 2 0 0 0 7 17.828V22",key:"1d314k"}],["path",{d:"M7 2v4.172a2 2 0 0 0 .586 1.414L12 12l4.414-4.414A2 2 0 0 0 17 6.172V2",key:"1vvvr6"}]]);/**
 * @license lucide-react v0.460.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const g=s("MessageCircle",[["path",{d:"M7.9 20A9 9 0 1 0 4 16.1L2 22Z",key:"vv11sd"}]]);/**
 * @license lucide-react v0.460.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const p=s("PhoneMissed",[["line",{x1:"22",x2:"16",y1:"2",y2:"8",key:"1xzwqn"}],["line",{x1:"16",x2:"22",y1:"2",y2:"8",key:"13zxdn"}],["path",{d:"M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z",key:"foiqr5"}]]);/**
 * @license lucide-react v0.460.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const b=s("Target",[["circle",{cx:"12",cy:"12",r:"10",key:"1mglay"}],["circle",{cx:"12",cy:"12",r:"6",key:"1vlfrh"}],["circle",{cx:"12",cy:"12",r:"2",key:"1c9p78"}]]);/**
 * @license lucide-react v0.460.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const j=s("Timer",[["line",{x1:"10",x2:"14",y1:"2",y2:"2",key:"14vaq8"}],["line",{x1:"12",x2:"15",y1:"14",y2:"11",key:"17fdiu"}],["circle",{cx:"12",cy:"14",r:"8",key:"1e1u0o"}]]);function f(e){if(!e||!e.answered)return;const a=r(e.sla_percent);return a>=80?"ok":a>=60?"warn":"danger"}function w(e){if(!e||!e.total)return;const a=r(e.abandon_rate);return a<=5?"ok":a<=10?"warn":"danger"}function M({k:e,s:a}){return n.jsxs("div",{className:"kpis",children:[n.jsx(l,{icon:n.jsx(y,{}),label:"Calls waiting",value:o(e.waiting),tone:e.waiting>0?e.available?"warn":"danger":void 0}),n.jsx(l,{icon:n.jsx(v,{}),label:"Longest wait",value:e.longestWait?t(e.longestWait):"—",tone:e.longestWait>120?"danger":e.longestWait>45?"warn":void 0}),n.jsx(l,{icon:n.jsx(x,{}),label:"Active calls",value:o(e.active)}),n.jsx(l,{icon:n.jsx(c,{}),label:"Agents available",value:o(e.available),tone:e.available?"ok":void 0,sub:`${e.loggedIn} logged in`}),n.jsx(l,{icon:n.jsx(g,{}),label:"On call / wrap-up",value:`${e.busy} / ${e.wrapup}`}),n.jsx(l,{icon:n.jsx(u,{}),label:"Paused",value:o(e.paused)}),n.jsx(l,{icon:n.jsx(d,{}),label:"AI sessions",value:o(e.aiSessions)}),n.jsx(l,{icon:n.jsx(b,{}),label:"SLA today",value:a&&a.answered?i(a.sla_percent):"—",tone:f(a),sub:a?`${o(a.sla_met)} of ${o(a.answered)} answered`:void 0}),n.jsx(l,{icon:n.jsx(p,{}),label:"Abandon rate",value:a&&a.total?i(a.abandon_rate):"—",tone:w(a),sub:a?`${o(a.abandoned)} of ${o(a.total)} calls`:void 0}),n.jsx(l,{icon:n.jsx(j,{}),label:"Avg wait / talk",value:a?`${t(a.avg_wait_sec)}`:"—",sub:a?`Talk ${t(a.avg_talk_sec)}`:void 0})]})}export{M as K,w as a,f as s};
