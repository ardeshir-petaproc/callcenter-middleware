import { useEffect, useState } from 'react';
import { Users } from 'lucide-react';
import { post } from '../../api/client';
import { useLiveAgents } from '../../state/selectors';
import { agentStatusLabel } from '../ui/Badges';
import { Dialog } from '../ui/Dialog';
import { Field, Segmented } from '../ui/primitives';
import { toast } from '../ui/Toast';

type Kind = 'agent' | 'external';

/** Starts a warm transfer: the caller is held while the chosen agent or number rings. */
export function ConsultDialog({ callId, agentId, open, onClose }: { callId: string; agentId: number; open: boolean; onClose: () => void }) {
  const [kind, setKind] = useState<Kind>('agent');
  const [target, setTarget] = useState('');
  const [busy, setBusy] = useState(false);
  const agents = useLiveAgents().filter((a) => a.user_id !== agentId);
  useEffect(() => { setTarget(''); }, [kind, open]);
  const submit = async () => {
    setBusy(true);
    try {
      await post(`/calls/${encodeURIComponent(callId)}/consult`, { type: kind, target });
      toast.success('Consulting — the caller is on hold');
      onClose();
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  const available = agents.filter((a) => a.status === 'available');
  return (
    <Dialog open={open} onClose={onClose} title="Consult before transferring"
      footer={<><button className="btn" onClick={onClose}>Cancel</button><button className="btn primary" disabled={busy || !target} onClick={submit}><Users />Start consult</button></>}>
      <p className="small muted">The caller is put on hold while you talk to the other party. Then complete the transfer, start a three-way conference, or cancel and return to the caller.</p>
      <Segmented<Kind> label="Consult" value={kind} onChange={setKind} options={[{ value: 'agent', label: 'Agent' }, { value: 'external', label: 'Number' }]} />
      {kind === 'agent' ? (
        <Field label="Agent" hint={available.length ? `${available.length} available` : 'No agent is available right now'}>
          <select className="select" value={target} onChange={(e) => setTarget(e.target.value)} data-autofocus>
            <option value="">Select an agent…</option>
            {[...agents].sort((a, b) => Number(b.status === 'available') - Number(a.status === 'available')).map((a) => (
              <option key={a.user_id} value={String(a.user_id)} disabled={a.status !== 'available'}>{a.display_name || a.username} — {agentStatusLabel(a.status)}</option>
            ))}
          </select>
        </Field>
      ) : (
        <Field label="Phone number"><input className="input" inputMode="tel" placeholder="+1 555 123 4567" value={target} onChange={(e) => setTarget(e.target.value)} data-autofocus /></Field>
      )}
    </Dialog>
  );
}
