import { useEffect, useState } from 'react';
import { Dialog } from '../ui/Dialog';
import { Field, Segmented } from '../ui/primitives';
import { toast } from '../ui/Toast';
import { post } from '../../api/client';
import { useLookup } from '../../state/lookups';
import { useLiveAgents } from '../../state/selectors';
import { agentStatusLabel } from '../ui/Badges';

type TType = 'queue' | 'agent' | 'external' | 'ai' | 'voicemail';
const TYPES: { value: TType; label: string }[] = [
  { value: 'queue', label: 'Queue' }, { value: 'agent', label: 'Agent' }, { value: 'external', label: 'Number' },
  { value: 'ai', label: 'AI agent' }, { value: 'voicemail', label: 'Voicemail' },
];

export function TransferDialog({ callId, open, onClose }: { callId: string; open: boolean; onClose: () => void }) {
  const [type, setType] = useState<TType>('queue');
  const [target, setTarget] = useState('');
  const [busy, setBusy] = useState(false);
  const queues = useLookup('queues');
  const profiles = useLookup('aiProfiles');
  const agents = useLiveAgents();

  useEffect(() => { setTarget(''); }, [type, open]);

  const submit = async () => {
    setBusy(true);
    try {
      await post(`/calls/${encodeURIComponent(callId)}/transfer`, { type, target });
      toast.success('Transfer started');
      onClose();
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };

  const needsTarget = type !== 'voicemail';
  return (
    <Dialog open={open} onClose={onClose} title="Transfer call"
      footer={<><button className="btn" onClick={onClose}>Cancel</button>
        <button className="btn primary" disabled={busy || (needsTarget && !target)} onClick={submit}>Transfer</button></>}>
      <Segmented label="Transfer type" value={type} options={TYPES} onChange={setType} />
      {type === 'queue' && (
        <Field label="Queue">
          <select className="select" value={target} onChange={(e) => setTarget(e.target.value)} data-autofocus>
            <option value="">Select a queue…</option>
            {queues.filter((q) => q.enabled).map((q) => <option key={q.id} value={String(q.id)}>{q.name}</option>)}
          </select>
        </Field>
      )}
      {type === 'agent' && (
        <Field label="Agent" hint="Available agents are listed first">
          <select className="select" value={target} onChange={(e) => setTarget(e.target.value)} data-autofocus>
            <option value="">Select an agent…</option>
            {[...agents].sort((a, b) => Number(b.status === 'available') - Number(a.status === 'available')).map((a) => (
              <option key={a.user_id} value={String(a.user_id)}>{a.display_name || a.username} — {agentStatusLabel(a.status)}</option>
            ))}
          </select>
        </Field>
      )}
      {type === 'external' && (
        <Field label="Phone number" hint="E.164 or dialable number, e.g. +15551234567">
          <input className="input" inputMode="tel" value={target} onChange={(e) => setTarget(e.target.value)} placeholder="+1 555 123 4567" data-autofocus />
        </Field>
      )}
      {type === 'ai' && (
        <Field label="AI profile">
          <select className="select" value={target} onChange={(e) => setTarget(e.target.value)} data-autofocus>
            <option value="">Select an AI agent…</option>
            {profiles.filter((p) => p.enabled).map((p) => <option key={p.id} value={String(p.id)}>{p.name}</option>)}
          </select>
        </Field>
      )}
      {type === 'voicemail' && (
        <Field label="Voicemail box (queue)" hint="Optional — leave empty for the call's current queue">
          <select className="select" value={target} onChange={(e) => setTarget(e.target.value)}>
            <option value="">Current queue</option>
            {queues.map((q) => <option key={q.id} value={String(q.id)}>{q.name}</option>)}
          </select>
        </Field>
      )}
    </Dialog>
  );
}
