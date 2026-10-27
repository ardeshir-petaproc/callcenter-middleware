import { useEffect, useState } from 'react';
import { Grid3x3, Mic, MicOff, Pause, Phone, PhoneCall, PhoneOff, Play, Settings2 } from 'lucide-react';
import { Card } from '../../components/ui/primitives';
import { Dialpad } from '../../components/ui/Dialpad';
import { toast } from '../../components/ui/Toast';
import { useNow, useStore } from '../../lib/store';
import { fmtDuration } from '../../lib/format';
import { useMe } from '../../state/auth';
import {
  answer, dial, dtmf, hangup, loadConfig, register, softphone, toggleHold, toggleMute, unregister, type SoftphoneConfig,
} from '../../state/softphone';
import { SoftphoneSettings } from './SoftphoneSettings';

const regBadge = {
  unregistered: ['', 'Not registered'], connecting: ['warn', 'Connecting…'], registered: ['ok', 'Registered'], failed: ['danger', 'Failed'],
} as const;

export function SoftphoneCard() {
  const me = useMe();
  const { reg, error, call } = useStore(softphone, (s) => s);
  const [cfg, setCfg] = useState<SoftphoneConfig | null>(() => loadConfig(me.id));
  const [settings, setSettings] = useState(false);
  const [pad, setPad] = useState(false);
  const [number, setNumber] = useState('');
  const now = useNow();

  useEffect(() => {
    if (cfg?.autoRegister && reg === 'unregistered') void register(cfg);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const [tone, label] = regBadge[reg];
  const placeCall = () => {
    try { dial(number.trim()); setNumber(''); } catch (e) { toast.error(e); }
  };

  return (
    <Card title="Softphone" icon={<Phone />} actions={
      <>
        <span className={`badge ${tone}`}><span className="dot" />{label}</span>
        <button className="btn ghost icon sm" onClick={() => setSettings(true)} aria-label="Softphone settings"><Settings2 /></button>
      </>
    }>
      {!cfg ? (
        <div className="col" style={{ alignItems: 'flex-start' }}>
          <p className="small muted">Use your browser as a SIP phone (WebRTC). Configure your WSS server and SIP credentials to get started.</p>
          <button className="btn" onClick={() => setSettings(true)}><Settings2 />Configure softphone</button>
        </div>
      ) : (
        <div className="col gap-lg">
          {error && <div className="alert danger small">{error}</div>}
          {call ? (
            <div className="col" style={{ alignItems: 'center', gap: 6 }}>
              <div className="small muted">{call.direction === 'incoming' ? (call.state === 'ringing' ? 'Incoming call' : 'Inbound') : 'Outbound'}</div>
              <div className="strong" style={{ fontSize: 18 }}>{call.remote}</div>
              <div className="num muted">{call.state === 'active' && call.startedAt ? fmtDuration((now - call.startedAt) / 1000) : call.state === 'ringing' ? 'Ringing…' : 'Connecting…'}</div>
              <div className="btn-group" style={{ justifyContent: 'center', marginTop: 6 }}>
                {call.state === 'ringing' && call.direction === 'incoming' ? (
                  <>
                    <button className="btn success" onClick={answer}><PhoneCall />Answer</button>
                    <button className="btn danger" onClick={hangup}><PhoneOff />Decline</button>
                  </>
                ) : (
                  <>
                    <button className={`btn ${call.muted ? 'active' : ''}`} onClick={toggleMute} aria-pressed={call.muted}>{call.muted ? <MicOff /> : <Mic />}{call.muted ? 'Unmute' : 'Mute'}</button>
                    <button className={`btn ${call.held ? 'active' : ''}`} onClick={toggleHold} disabled={call.state !== 'active'}>{call.held ? <Play /> : <Pause />}{call.held ? 'Resume' : 'Hold'}</button>
                    <button className={`btn ${pad ? 'active' : ''}`} onClick={() => setPad((p) => !p)} aria-label="Keypad"><Grid3x3 /></button>
                    <button className="btn danger" onClick={hangup}><PhoneOff />Hang up</button>
                  </>
                )}
              </div>
              {pad && call.state === 'active' && <div style={{ width: '100%', maxWidth: 260, marginTop: 8 }}><Dialpad onPress={dtmf} /></div>}
            </div>
          ) : reg === 'registered' ? (
            <form className="row" onSubmit={(e) => { e.preventDefault(); placeCall(); }}>
              <input className="input" inputMode="tel" placeholder="Extension or number" value={number} onChange={(e) => setNumber(e.target.value)} aria-label="Number to call via softphone" />
              <button className="btn success" disabled={!number.trim()} type="submit"><PhoneCall />Call</button>
            </form>
          ) : (
            <div className="small muted ellipsis">{cfg.sipUri}</div>
          )}
          <div className="row between">
            {reg === 'registered' || reg === 'connecting'
              ? <button className="btn sm" onClick={() => void unregister()}>Unregister</button>
              : <button className="btn sm primary" onClick={() => void register(cfg)}>Register</button>}
            <span className="small muted">Remote audio plays through this browser</span>
          </div>
        </div>
      )}
      <SoftphoneSettings open={settings} onClose={() => setSettings(false)} initial={cfg} onSaved={(c) => { setCfg(c); void register(c); }} />
    </Card>
  );
}
