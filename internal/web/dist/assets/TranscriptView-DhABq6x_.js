import{c as l,j as a,E as o,ak as h}from"./index-DpvNHzdC.js";import{r as n}from"./react-5b195e9U.js";/**
 * @license lucide-react v0.460.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const p=l("MessageSquare",[["path",{d:"M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z",key:"1lielz"}]]);/**
 * @license lucide-react v0.460.0 - ISC
 *
 * This source code is licensed under the ISC license.
 * See the LICENSE file in the root directory of this source tree.
 */const u=l("PhoneOff",[["path",{d:"M10.68 13.31a16 16 0 0 0 3.41 2.6l1.27-1.27a2 2 0 0 1 2.11-.45 12.84 12.84 0 0 0 2.81.7 2 2 0 0 1 1.72 2v3a2 2 0 0 1-2.18 2 19.79 19.79 0 0 1-8.63-3.07 19.42 19.42 0 0 1-3.33-2.67m-2.67-3.34a19.79 19.79 0 0 1-3.07-8.63A2 2 0 0 1 4.11 2h3a2 2 0 0 1 2 1.72 12.84 12.84 0 0 0 .7 2.81 2 2 0 0 1-.45 2.11L8.09 9.91",key:"z86iuo"}],["line",{x1:"22",x2:"2",y1:"2",y2:"22",key:"11kh81"}]]),d={caller:"left",ai:"right",agent:"right agent",call:"center"},c={caller:"Caller",ai:"AI agent",agent:"Agent",call:""};function x({entries:s,follow:t}){const r=n.useRef(null);return n.useEffect(()=>{var e;t&&((e=r.current)==null||e.scrollIntoView({block:"nearest",behavior:"smooth"}))},[s.length,t]),s.length?a.jsxs("div",{className:"transcript",children:[s.map((e,i)=>a.jsxs("div",{className:`bubble-row ${d[e.speaker]??"left"}`,children:[a.jsx("div",{className:"bubble",children:e.text}),a.jsxs("div",{className:"bubble-meta",children:[c[e.speaker]??e.speaker,c[e.speaker]?" · ":"",h(e.at)]})]},`${e.at}-${i}`)),a.jsx("div",{ref:r})]}):a.jsx(o,{title:"No transcript yet",icon:a.jsx(p,{}),children:"Transcribed speech appears here as the conversation happens."})}export{p as M,u as P,x as T};
