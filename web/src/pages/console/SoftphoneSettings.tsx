import { useEffect, useState } from 'react';
import { Dialog } from '../../components/ui/Dialog';
import { Field, Toggle } from '../../components/ui/primitives';
import { useMe } from '../../state/auth';
import { saveConfig, type SoftphoneConfig } from '../../state/softphone';

interface Props { open: boolean; onClose: () => void; initial: SoftphoneConfig | null; onSaved: (c: SoftphoneConfig) => void }

export function SoftphoneSettings({ open, onClose, initial, onSaved }: Props) {
  const me = useMe();
  const host = typeof location !== 'undefined' ? location.hostname : 'pbx.example.com';
  const blank: SoftphoneConfig = {
    wssUrl: `wss://${host}:8089/ws`, sipUri: `sip:${me.extension || me.username}@${host}`, password: '',
    displayName: me.display_name || me.username, autoRegister: false,
  };
  const [cfg, setCfg] = useState<SoftphoneConfig>(initial ?? blank);
  useEffect(() => { if (open) setCfg(initial ?? blank); /* eslint-disable-next-line react-hooks/exhaustive-deps */ }, [open]);
  const set = (p: Partial<SoftphoneConfig>) => setCfg((c) => ({ ...c, ...p }));
  const valid = /^wss?:\/\//.test(cfg.wssUrl) && /^sips?:/.test(cfg.sipUri);

  return (
    <Dialog open={open} onClose={onClose} title="Softphone settings"
      footer={<><button className="btn" onClick={onClose}>Cancel</button>
        <button className="btn primary" disabled={!valid} onClick={() => { saveConfig(me.id, cfg); onSaved(cfg); onClose(); }}>Save & register</button></>}>
      <p className="small muted">Stored only in this browser for your user. Your PBX must expose SIP over secure WebSocket (e.g. Asterisk <code>pjsip</code> with a <code>transport-wss</code>).</p>
      <Field label="WebSocket (WSS) URL" hint="e.g. wss://pbx.example.com:8089/ws">
        <input className="input" value={cfg.wssUrl} onChange={(e) => set({ wssUrl: e.target.value })} />
      </Field>
      <Field label="SIP URI" hint="e.g. sip:1001@pbx.example.com">
        <input className="input" value={cfg.sipUri} onChange={(e) => set({ sipUri: e.target.value })} />
      </Field>
      <Field label="SIP password">
        <input className="input" type="password" autoComplete="off" value={cfg.password} onChange={(e) => set({ password: e.target.value })} />
      </Field>
      <Field label="Display name">
        <input className="input" value={cfg.displayName} onChange={(e) => set({ displayName: e.target.value })} />
      </Field>
      <Toggle checked={!!cfg.autoRegister} onChange={(v) => set({ autoRegister: v })} label="Register automatically when I open the console" />
    </Dialog>
  );
}
