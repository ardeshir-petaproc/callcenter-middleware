import { useEffect, useRef, useState } from 'react';
import { PhoneOff } from 'lucide-react';
import type { AIProfile, ClientSecret, TranscriptEntry } from '../../api/types';
import { errMsg, post } from '../../api/client';
import { Dialog } from '../../components/ui/Dialog';
import { TranscriptView } from '../../components/calls/TranscriptView';
import { startRealtime, type RealtimeSession } from './realtime';

type St = 'idle' | 'connecting' | 'live' | 'ended' | 'error';
const label: Record<St, string> = { idle: 'Starting…', connecting: 'Connecting…', live: 'Live — speak now', ended: 'Call ended', error: 'Error' };

/** "Test in browser": mint an ephemeral key and talk to the AI profile over WebRTC. */
export function AITestDialog({ profile, onClose }: { profile: AIProfile; onClose: () => void }) {
  const [state, setState] = useState<St>('idle');
  const [error, setError] = useState<string | null>(null);
  const [lines, setLines] = useState<TranscriptEntry[]>([]);
  const [speaking, setSpeaking] = useState(false);
  const sess = useRef<RealtimeSession | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const secret = await post<ClientSecret>(`/ai/profiles/${profile.id}/client-secret`);
        if (cancelled) return;
        sess.current = await startRealtime(secret, {
          onLine: (l) => setLines((x) => [...x, { call_id: 'test', speaker: l.speaker, text: l.text, at: l.at }]),
          onState: (s, d) => { setState(s); if (d) setError(d); },
          onSpeaking: setSpeaking,
        });
        if (cancelled) sess.current.hangup();
      } catch (e) {
        if (!cancelled) { setState('error'); setError((prev) => prev ?? errMsg(e)); }
      }
    })();
    return () => { cancelled = true; sess.current?.hangup(); };
  }, [profile.id]);

  const live = state === 'live' || state === 'connecting' || state === 'idle';
  return (
    <Dialog open title={`Test “${profile.name}”`} onClose={onClose}
      footer={live
        ? <button className="btn danger" onClick={() => { sess.current?.hangup(); setState('ended'); }}><PhoneOff />Hang up</button>
        : <button className="btn" onClick={onClose}>Close</button>}>
      <div className="col" style={{ alignItems: 'center', gap: 10 }}>
        <div className={`ai-orb ${state === 'live' && speaking ? 'live' : ''}`} style={state !== 'live' ? { opacity: 0.45 } : undefined} aria-hidden />
        <span className={`badge ${state === 'live' ? 'ok' : state === 'error' ? 'danger' : ''}`}><span className="dot" />{label[state]}</span>
        <span className="small muted">Uses your microphone. Model: {profile.model || 'default'} · Voice: {profile.voice || 'default'}</span>
      </div>
      {error && <div className="alert danger small pre">{error}</div>}
      <div style={{ maxHeight: 320, overflowY: 'auto', border: '1px solid var(--border)', borderRadius: 8, padding: 8 }}>
        <TranscriptView entries={lines} follow />
      </div>
    </Dialog>
  );
}
