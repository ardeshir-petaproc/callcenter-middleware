import { API_BASE, get, getToken } from '../api/client';
import type { AgentState, Call, CallView, ScreenPop, Snapshot, TranscriptEntry, WsMessage } from '../api/types';
import {
  applyAgent, applyCallEnded, applyCallUpdated, applyScreenPop, applySnapshot, applyTranscript, live, noteServerTime,
} from './live';

type Listener = (data: unknown, type: string) => void;
const listeners = new Map<string, Set<Listener>>();

/** Subscribe to raw realtime events by type ('*' for all). */
export function onLive(type: string, fn: Listener): () => void {
  if (!listeners.has(type)) listeners.set(type, new Set());
  listeners.get(type)!.add(fn);
  return () => { listeners.get(type)?.delete(fn); };
}

function emit(type: string, data: unknown) {
  listeners.get(type)?.forEach((fn) => fn(data, type));
  listeners.get('*')?.forEach((fn) => fn(data, type));
}

let ws: WebSocket | null = null;
let retry = 0;
let retryTimer: number | undefined;
let stopped = true;

function handle(msg: WsMessage) {
  switch (msg.type) {
    case 'snapshot':
      noteServerTime(msg.at || (msg.data as Snapshot).at);
      applySnapshot(msg.data as Snapshot);
      break;
    case 'call.updated': applyCallUpdated(msg.data as CallView); break;
    case 'call.ended': applyCallEnded(msg.data as Call); break;
    case 'agent.updated': applyAgent(msg.data as AgentState); break;
    case 'call.transcript': applyTranscript(msg.data as TranscriptEntry); break;
    case 'agent.screenpop': applyScreenPop(msg.data as ScreenPop); break;
    case 'voicemail.created': live.set((s) => ({ unheardVoicemails: s.unheardVoicemails + 1 })); break;
    default: break;
  }
  emit(msg.type, msg.data);
}

async function pollOnce() {
  try { applySnapshot(await get<Snapshot>('/live')); } catch { /* 401 handled globally */ }
}

function connect() {
  retryTimer = undefined;
  const token = getToken();
  if (stopped || !token) return;
  live.set({ conn: 'connecting' });
  const proto = location.protocol === 'https:' ? 'wss' : 'ws';
  const sock = new WebSocket(`${proto}://${location.host}${API_BASE}/ws?token=${encodeURIComponent(token)}`);
  ws = sock;
  sock.onopen = () => { retry = 0; live.set({ conn: 'open' }); };
  sock.onmessage = (ev) => {
    try { handle(JSON.parse(ev.data as string) as WsMessage); } catch (e) { console.warn('bad ws message', e); }
  };
  sock.onclose = () => {
    if (ws !== sock) return;
    ws = null;
    if (stopped) { live.set({ conn: 'idle' }); return; }
    live.set({ conn: 'closed' });
    scheduleReconnect();
  };
  sock.onerror = () => { /* onclose follows */ };
}

function scheduleReconnect() {
  retry += 1;
  const base = Math.min(30000, 1000 * 2 ** Math.min(retry - 1, 5));
  const delay = base / 2 + Math.random() * (base / 2);
  // Keep data fresh (and detect an expired token via the REST 401 path) while the socket is down.
  void pollOnce();
  retryTimer = window.setTimeout(connect, delay);
}

export function startLive() {
  stopped = false;
  if (ws || retryTimer !== undefined) stopLiveSocket();
  stopped = false;
  void pollOnce();
  connect();
}

function stopLiveSocket() {
  if (retryTimer !== undefined) { clearTimeout(retryTimer); retryTimer = undefined; }
  if (ws) { const s = ws; ws = null; s.close(); }
}

export function stopLive() {
  stopped = true;
  stopLiveSocket();
  live.set({ conn: 'idle' });
}

/** Force an immediate reconnect (e.g. from the connection indicator). */
export function reconnectLive() {
  retry = 0;
  stopLiveSocket();
  stopped = false;
  connect();
}
