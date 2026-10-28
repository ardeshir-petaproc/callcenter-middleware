import{c as l,j as n,aM as t,L as o,f as i,$ as r,v as u,N as s,aa as v,l as d,aL as c}from"./index-CYNLLgRk.js";import{C as x}from"./clock-Cz-HAJ-O.js";import{P as g}from"./phone-call-BoVWggsH.js";import{M as y}from"./message-circle-L59neQDk.js";import{C as b}from"./coffee-CqIU0opn.js";/**
 * @license lucide-react v0.460.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const p=l("Hourglass",[["path",{d:"M5 22h14",key:"ehvnwv"}],["path",{d:"M5 2h14",key:"pdyrp9"}],["path",{d:"M17 22v-4.172a2 2 0 0 0-.586-1.414L12 12l-4.414 4.414A2 2 0 0 0 7 17.828V22",key:"1d314k"}],["path",{d:"M7 2v4.172a2 2 0 0 0 .586 1.414L12 12l4.414-4.414A2 2 0 0 0 17 6.172V2",key:"1vvvr6"}]]);/**
 * @license lucide-react v0.460.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const f=l("PhoneMissed",[["line",{x1:"22",x2:"16",y1:"2",y2:"8",key:"1xzwqn"}],["line",{x1:"16",x2:"22",y1:"2",y2:"8",key:"13zxdn"}],["path",{d:"M22 16.92v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.5 19.5 0 0 1-6-6 19.79 19.79 0 0 1-3.07-8.67A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91a16 16 0 0 0 6 6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7A2 2 0 0 1 22 16.92z",key:"foiqr5"}]]);/**
 * @license lucide-react v0.460.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const j=l("Target",[["circle",{cx:"12",cy:"12",r:"10",key:"1mglay"}],["circle",{cx:"12",cy:"12",r:"6",key:"1vlfrh"}],["circle",{cx:"12",cy:"12",r:"2",key:"1c9p78"}]]);/**
 * @license lucide-react v0.460.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const m=l("Timer",[["line",{x1:"10",x2:"14",y1:"2",y2:"2",key:"14vaq8"}],["line",{x1:"12",x2:"15",y1:"14",y2:"11",key:"17fdiu"}],["circle",{cx:"12",cy:"14",r:"8",key:"1e1u0o"}]]);function w(e){if(!e||!e.answered)return;const a=c(e.sla_percent);return a>=80?"ok":a>=60?"warn":"danger"}function _(e){if(e!=null&&e.csat_count)return e.csat_avg>=4?"ok":e.csat_avg>=3?"warn":"danger"}function h(e){if(!e||!e.total)return;const a=c(e.abandon_rate);return a<=5?"ok":a<=10?"warn":"danger"}function C({k:e,s:a}){return n.jsxs("div",{className:"kpis",children:[n.jsx(t,{icon:n.jsx(p,{}),label:"Calls waiting",value:o(e.waiting),tone:e.waiting>0?e.available?"warn":"danger":void 0}),n.jsx(t,{icon:n.jsx(x,{}),label:"Longest wait",value:e.longestWait?i(e.longestWait):"—",tone:e.longestWait>120?"danger":e.longestWait>45?"warn":void 0}),n.jsx(t,{icon:n.jsx(g,{}),label:"Active calls",value:o(e.active)}),n.jsx(t,{icon:n.jsx(r,{}),label:"Agents available",value:o(e.available),tone:e.available?"ok":void 0,sub:`${e.loggedIn} logged in`}),n.jsx(t,{icon:n.jsx(y,{}),label:"On call / wrap-up",value:`${e.busy} / ${e.wrapup}`}),n.jsx(t,{icon:n.jsx(b,{}),label:"Paused",value:o(e.paused)}),n.jsx(t,{icon:n.jsx(u,{}),label:"AI sessions",value:o(e.aiSessions)}),n.jsx(t,{icon:n.jsx(j,{}),label:"SLA today",value:a&&a.answered?s(a.sla_percent):"—",tone:w(a),sub:a?`${o(a.sla_met)} of ${o(a.answered)} answered`:void 0}),n.jsx(t,{icon:n.jsx(f,{}),label:"Abandon rate",value:a&&a.total?s(a.abandon_rate):"—",tone:h(a),sub:a?`${o(a.abandoned)} of ${o(a.total)} calls`:void 0}),n.jsx(t,{icon:n.jsx(d,{}),label:"CSAT today",value:a?v(a.csat_avg,a.csat_count):"—",tone:_(a),sub:a!=null&&a.csat_count?`${o(a.csat_count)} rating${a.csat_count===1?"":"s"} · out of 5`:"No ratings yet"}),n.jsx(t,{icon:n.jsx(m,{}),label:"Avg wait / talk",value:a?`${i(a.avg_wait_sec)}`:"—",sub:a?`Talk ${i(a.avg_talk_sec)}`:void 0})]})}export{C as K,h as a,_ as c,w as s};
