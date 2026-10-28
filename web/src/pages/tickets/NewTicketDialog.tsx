import { useState } from 'react';
import { post } from '../../api/client';
import { TICKET_PRIORITIES, type Ticket, type TicketInput } from '../../api/types_work';
import { humanize } from '../../lib/format';
import { useLookup } from '../../state/lookups';
import { Dialog } from '../../components/ui/Dialog';
import { Field } from '../../components/ui/primitives';
import { TagInput } from '../../components/ui/TagInput';
import { toast } from '../../components/ui/Toast';

const blank: TicketInput = {
  subject: '', description: '', category: '', priority: 'normal', department_id: 0, assignee_id: 0,
  requester_name: '', requester_phone: '', requester_email: '', tags: [],
};

export function NewTicketDialog({ open, onClose, onCreated }: { open: boolean; onClose: () => void; onCreated: (t: Ticket) => void }) {
  const [t, setT] = useState<TicketInput>(blank);
  const [busy, setBusy] = useState(false);
  const depts = useLookup('departments');
  const users = useLookup('users');
  const set = (p: TicketInput) => setT((x) => ({ ...x, ...p }));
  const close = () => { setT(blank); onClose(); };
  const submit = async () => {
    setBusy(true);
    try {
      const created = await post<Ticket>('/tickets', { ...t, source: 'manual' });
      toast.success(`Ticket ${created.number} created`);
      close();
      onCreated(created);
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  return (
    <Dialog open={open} onClose={close} title="New ticket" size="wide"
      footer={<><button className="btn" onClick={close}>Cancel</button><button className="btn primary" disabled={busy || !t.subject?.trim()} onClick={submit}>Create ticket</button></>}>
      <Field label="Subject"><input className="input" dir="auto" value={t.subject} onChange={(e) => set({ subject: e.target.value })} data-autofocus /></Field>
      <Field label="Description"><textarea className="textarea" dir="auto" value={t.description} onChange={(e) => set({ description: e.target.value })} /></Field>
      <div className="form-grid">
        <Field label="Priority">
          <select className="select" value={t.priority} onChange={(e) => set({ priority: e.target.value as Ticket['priority'] })}>
            {TICKET_PRIORITIES.map((p) => <option key={p} value={p}>{humanize(p)}</option>)}
          </select>
        </Field>
        <Field label="Category"><input className="input" dir="auto" placeholder="e.g. billing" value={t.category} onChange={(e) => set({ category: e.target.value })} /></Field>
        <Field label="Department">
          <select className="select" value={t.department_id || ''} onChange={(e) => set({ department_id: Number(e.target.value) || 0 })}>
            <option value="">None</option>
            {depts.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
          </select>
        </Field>
        <Field label="Assignee">
          <select className="select" value={t.assignee_id || ''} onChange={(e) => set({ assignee_id: Number(e.target.value) || 0 })}>
            <option value="">Unassigned</option>
            {users.map((u) => <option key={u.id} value={u.id}>{u.display_name || u.username}</option>)}
          </select>
        </Field>
        <Field label="Requester name"><input className="input" dir="auto" value={t.requester_name} onChange={(e) => set({ requester_name: e.target.value })} /></Field>
        <Field label="Requester phone"><input className="input" inputMode="tel" value={t.requester_phone} onChange={(e) => set({ requester_phone: e.target.value })} /></Field>
        <Field label="Requester email"><input className="input" type="email" value={t.requester_email} onChange={(e) => set({ requester_email: e.target.value })} /></Field>
        <div className="field span-all"><span className="field-label">Tags</span><TagInput value={t.tags ?? []} onChange={(v) => set({ tags: v })} /></div>
      </div>
    </Dialog>
  );
}
