import type { AgentState, Call, CallView, QueueAlert, QueueView, Role, ScreenPop, Snapshot, TranscriptEntry } from '../api/types';
import { createStore, setServerOffset } from '../lib/store';

export type ConnState = 'idle' | 'connecting' | 'open' | 'closed';

export interface LiveState {
  conn: ConnState;
  hasSnapshot: boolean;
  calls: Record<string, CallView>;
  agents: Record<number, AgentState>;
  queues: QueueView[];
  aiSessions: number;
  snapshotAt: string | null;
  transcripts: Record<string, TranscriptEntry[]>;
  /** Screen-pop targeting the logged-in user (ringing). */
  screenpop: ScreenPop | null;
  /** Last call that ended for the logged-in user and still needs wrap-up. */
  wrapup: Call | null;
  unheardVoicemails: number;
  /** Queues currently over their alert thresholds (from `queue.alert`). */
  queueAlerts: Record<number, QueueAlert>;
  meId: number;
  meRole: Role | null;
}

const initial: LiveState = {
  conn: 'idle', hasSnapshot: false, calls: {}, agents: {}, queues: [], aiSessions: 0, snapshotAt: null,
  transcripts: {}, screenpop: null, wrapup: null, unheardVoicemails: 0, queueAlerts: {}, meId: 0, meRole: null,
};

export const live = createStore<LiveState>(initial);
const dismissedWrapups = new Set<string>();

export function resetLive(meId: number, meRole: Role) {
  dismissedWrapups.clear();
  live.set({ ...initial, meId, meRole });
}

export function applySnapshot(s: Snapshot) {
  const calls: Record<string, CallView> = {};
  for (const c of s.calls ?? []) calls[c.id] = c;
  const agents: Record<number, AgentState> = {};
  for (const a of s.agents ?? []) agents[a.user_id] = a;
  live.set((st) => ({
    hasSnapshot: true, calls, agents, queues: s.queues ?? [], aiSessions: s.ai_sessions ?? 0, snapshotAt: s.at,
    queueAlerts: s.alerts ? Object.fromEntries(s.alerts.map((a) => [a.queue_id, a])) : st.queueAlerts,
    screenpop: st.screenpop && calls[st.screenpop.call.id] && stillRingingMe(calls[st.screenpop.call.id], st.meId)
      ? { ...st.screenpop, call: calls[st.screenpop.call.id] } : null,
  }));
}

function stillRingingMe(c: CallView, me: number): boolean {
  if (c.state === 'ended') return false;
  if (c.agent_id === me && (c.state === 'connected' || c.state === 'hold')) return false;
  if (Array.isArray(c.ringing_agents)) return c.ringing_agents.includes(me);
  return c.state === 'ringing';
}

export function applyCallUpdated(c: CallView) {
  if (c.state === 'ended') { applyCallEnded(c); return; }
  live.set((st) => {
    let screenpop = st.screenpop;
    if (screenpop && screenpop.call.id === c.id) {
      screenpop = stillRingingMe(c, st.meId) ? { ...screenpop, call: c } : null;
    }
    return { calls: { ...st.calls, [c.id]: c }, screenpop };
  });
}

export function applyCallEnded(c: Call | CallView) {
  live.set((st) => {
    const calls = { ...st.calls };
    delete calls[c.id];
    const screenpop = st.screenpop?.call.id === c.id ? null : st.screenpop;
    let wrapup = st.wrapup;
    if (c.agent_id === st.meId && st.meId && c.answered_at && !dismissedWrapups.has(c.id)) {
      // A re-sent final CDR (after AI analysis) refreshes data but must not re-open a finished wrap-up.
      if (!wrapup || wrapup.id === c.id) wrapup = { ...(wrapup ?? {}), ...c } as Call;
    }
    return { calls, screenpop, wrapup };
  });
}

export function dismissWrapup(id: string) {
  dismissedWrapups.add(id);
  live.set((st) => (st.wrapup?.id === id ? { wrapup: null } : {}));
}

export function applyAgent(a: AgentState) {
  live.set((st) => ({ agents: { ...st.agents, [a.user_id]: a } }));
}

export function applyTranscript(t: TranscriptEntry) {
  live.set((st) => {
    const list = st.transcripts[t.call_id] ?? [];
    if (list.some((x) => x.at === t.at && x.text === t.text && x.speaker === t.speaker)) return {};
    const next = [...list, t].slice(-500);
    return { transcripts: { ...st.transcripts, [t.call_id]: next } };
  });
}

export function seedTranscript(callId: string, entries: TranscriptEntry[]) {
  live.set((st) => {
    const existing = st.transcripts[callId] ?? [];
    const merged = [...entries];
    for (const e of existing) {
      if (!merged.some((x) => x.at === e.at && x.text === e.text)) merged.push(e);
    }
    merged.sort((a, b) => Date.parse(a.at) - Date.parse(b.at));
    return { transcripts: { ...st.transcripts, [callId]: merged } };
  });
}

export function applyScreenPop(p: ScreenPop): boolean {
  const me = live.get().meId;
  if (p.user_id !== me) return false;
  live.set((st) => ({ screenpop: p, calls: { ...st.calls, [p.call.id]: p.call } }));
  return true;
}

export function applyQueueAlert(a: QueueAlert) {
  live.set((st) => {
    const queueAlerts = { ...st.queueAlerts };
    if (a.level === 'alert') queueAlerts[a.queue_id] = a;
    else delete queueAlerts[a.queue_id];
    return { queueAlerts };
  });
}

export function clearScreenPop() { live.set({ screenpop: null }); }

export function noteServerTime(at: string | undefined) {
  if (!at) return;
  const t = Date.parse(at);
  if (!Number.isNaN(t)) setServerOffset(t - Date.now());
}
