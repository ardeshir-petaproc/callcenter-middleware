// WebRTC softphone built on JsSIP (lazy-loaded). Fully optional: nothing runs until the user registers.
import type { UA } from 'jssip/lib/UA';
import type { RTCSession } from 'jssip/lib/RTCSession';
import { createStore } from '../lib/store';
import { startRinging, stopRinging } from '../lib/ringtone';

export interface SoftphoneConfig { wssUrl: string; sipUri: string; password: string; displayName: string; autoRegister?: boolean }
export type RegState = 'unregistered' | 'connecting' | 'registered' | 'failed';
export interface SoftCall {
  direction: 'incoming' | 'outgoing';
  remote: string;
  state: 'ringing' | 'connecting' | 'active';
  muted: boolean;
  held: boolean;
  startedAt?: number;
}
export interface SoftphoneState { reg: RegState; error: string | null; call: SoftCall | null }

export const softphone = createStore<SoftphoneState>({ reg: 'unregistered', error: null, call: null });

let ua: UA | null = null;
let session: RTCSession | null = null;
let audioEl: HTMLAudioElement | null = null;

function remoteAudio(): HTMLAudioElement {
  if (!audioEl) {
    audioEl = document.createElement('audio');
    audioEl.autoplay = true;
    audioEl.setAttribute('playsinline', '');
    audioEl.id = 'softphone-remote-audio';
    document.body.appendChild(audioEl);
  }
  return audioEl;
}

export function configKey(userId: number) { return `cc.softphone.${userId}`; }

export function loadConfig(userId: number): SoftphoneConfig | null {
  try {
    const raw = localStorage.getItem(configKey(userId));
    return raw ? (JSON.parse(raw) as SoftphoneConfig) : null;
  } catch { return null; }
}
export function saveConfig(userId: number, cfg: SoftphoneConfig) {
  try { localStorage.setItem(configKey(userId), JSON.stringify(cfg)); } catch { /* ignore */ }
}

function patchCall(p: Partial<SoftCall>) {
  softphone.set((s) => (s.call ? { call: { ...s.call, ...p } } : {}));
}

function attach(s: RTCSession, direction: 'incoming' | 'outgoing') {
  session = s;
  const remote = s.remote_identity?.display_name || s.remote_identity?.uri?.user || 'Unknown';
  softphone.set({ call: { direction, remote, state: direction === 'incoming' ? 'ringing' : 'connecting', muted: false, held: false } });
  if (direction === 'incoming') startRinging();
  const bindTrack = (pc: RTCPeerConnection) => {
    pc.addEventListener('track', (ev) => {
      const el = remoteAudio();
      el.srcObject = ev.streams[0] ?? new MediaStream([ev.track]);
      void el.play().catch(() => undefined);
    });
  };
  if (s.connection) bindTrack(s.connection as unknown as RTCPeerConnection);
  s.on('peerconnection', (e: { peerconnection: RTCPeerConnection }) => bindTrack(e.peerconnection));
  s.on('accepted', () => { stopRinging(); patchCall({ state: 'active', startedAt: Date.now() }); });
  s.on('confirmed', () => { stopRinging(); patchCall({ state: 'active' }); });
  const end = () => {
    stopRinging();
    if (session === s) session = null;
    softphone.set({ call: null });
    if (audioEl) audioEl.srcObject = null;
  };
  s.on('ended', end);
  s.on('failed', end);
}

export async function register(cfg: SoftphoneConfig) {
  await unregister();
  if (!cfg.wssUrl || !cfg.sipUri) { softphone.set({ reg: 'failed', error: 'WSS URL and SIP URI are required' }); return; }
  softphone.set({ reg: 'connecting', error: null });
  try {
    const JsSIP = await import('jssip');
    const socket = new JsSIP.WebSocketInterface(cfg.wssUrl);
    const agent = new JsSIP.UA({
      sockets: [socket], uri: cfg.sipUri, password: cfg.password, display_name: cfg.displayName || undefined,
      register: true, session_timers: false,
    });
    ua = agent;
    agent.on('registered', () => softphone.set({ reg: 'registered', error: null }));
    agent.on('unregistered', () => { if (ua === agent) softphone.set({ reg: 'unregistered' }); });
    agent.on('registrationFailed', (e: { cause?: string }) => softphone.set({ reg: 'failed', error: `Registration failed${e?.cause ? `: ${e.cause}` : ''}` }));
    agent.on('disconnected', () => { if (ua === agent && softphone.get().reg !== 'failed') softphone.set({ reg: 'connecting', error: 'Socket disconnected — retrying…' }); });
    agent.on('newRTCSession', (e: { originator: string; session: RTCSession }) => {
      if (session && e.originator === 'remote') { e.session.terminate({ status_code: 486, reason_phrase: 'Busy Here' }); return; }
      attach(e.session, e.originator === 'remote' ? 'incoming' : 'outgoing');
    });
    agent.start();
  } catch (e) {
    ua = null;
    softphone.set({ reg: 'failed', error: e instanceof Error ? e.message : String(e) });
  }
}

export async function unregister() {
  const agent = ua;
  ua = null;
  if (session) { try { session.terminate(); } catch { /* ignore */ } session = null; }
  if (agent) { try { agent.unregister({ all: false }); agent.stop(); } catch { /* ignore */ } }
  stopRinging();
  softphone.set({ reg: 'unregistered', error: null, call: null });
}

const media = { mediaConstraints: { audio: true, video: false } };

export function answer() { session?.answer(media); }
export function hangup() { try { session?.terminate(); } catch { /* already ended */ } }
export function toggleMute() {
  const s = session; const c = softphone.get().call;
  if (!s || !c) return;
  if (c.muted) s.unmute({ audio: true }); else s.mute({ audio: true });
  patchCall({ muted: !c.muted });
}
export function toggleHold() {
  const s = session; const c = softphone.get().call;
  if (!s || !c) return;
  if (c.held) s.unhold(); else s.hold();
  patchCall({ held: !c.held });
}
export function dtmf(tone: string) { try { session?.sendDTMF(tone); } catch { /* not established */ } }
export function dial(target: string) {
  if (!ua || softphone.get().reg !== 'registered') throw new Error('Softphone is not registered');
  ua.call(target, media);
}
