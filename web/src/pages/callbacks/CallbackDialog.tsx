import { useEffect, useState } from 'react';
import type { Callback, CallbackStatus } from '../../api/types';
import { post, put } from '../../api/client';
import { Dialog } from '../../components/ui/Dialog';
import { Field } from '../../components/ui/primitives';
import { toast } from '../../components/ui/Toast';
import { useLookup } from '../../state/lookups';

const STATUSES: CallbackStatus[] = ['pending', 'in_progress', 'done', 'failed', 'cancelled'];

function toLocalInput(iso?: string): string {
  if (!iso) return '';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime()) || d.getFullYear() < 1971) return '';
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
}

interface Props { open: boolean; initial: Callback | null; onClose: () => void; onSaved: (c: Callback) => void }

export function CallbackDialog({ open, initial, onClose, onSaved }: Props) {
  const queues = useLookup('queues');
  const users = useLookup('users');
  const [f, setF] = useState({ number: '', name: '', queue_id: 0, notes: '', due: '', status: 'pending' as CallbackStatus, assigned_to: 0 });
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    if (!open) return;
    setF(initial
      ? { number: initial.number, name: initial.name, queue_id: initial.queue_id, notes: initial.notes, due: toLocalInput(initial.due_at), status: initial.status, assigned_to: initial.assigned_to }
      : { number: '', name: '', queue_id: 0, notes: '', due: '', status: 'pending', assigned_to: 0 });
  }, [open, initial]);
  const set = (p: Partial<typeof f>) => setF((x) => ({ ...x, ...p }));

  const save = async () => {
    setBusy(true);
    try {
      const due_at = f.due ? new Date(f.due).toISOString() : undefined;
      const saved = initial
        ? await put<Callback>(`/callbacks/${initial.id}`, { ...initial, number: f.number, name: f.name, queue_id: f.queue_id, notes: f.notes, due_at, status: f.status, assigned_to: f.assigned_to })
        : await post<Callback>('/callbacks', { number: f.number, name: f.name || undefined, queue_id: f.queue_id || undefined, notes: f.notes || undefined, due_at });
      toast.success(initial ? 'Callback updated' : 'Callback scheduled');
      onSaved(saved);
      onClose();
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };

  return (
    <Dialog open={open} onClose={onClose} title={initial ? 'Edit callback' : 'New callback'}
      footer={<><button className="btn" onClick={onClose}>Cancel</button><button className="btn primary" disabled={busy || !f.number.trim()} onClick={save}>Save</button></>}>
      <div className="form-grid">
        <Field label="Phone number"><input className="input" inputMode="tel" value={f.number} onChange={(e) => set({ number: e.target.value })} data-autofocus /></Field>
        <Field label="Name"><input className="input" value={f.name} onChange={(e) => set({ name: e.target.value })} /></Field>
        <Field label="Queue">
          <select className="select" value={f.queue_id || ''} onChange={(e) => set({ queue_id: Number(e.target.value) || 0 })}>
            <option value="">Any</option>
            {queues.map((q) => <option key={q.id} value={q.id}>{q.name}</option>)}
          </select>
        </Field>
        <Field label="Due"><input className="input" type="datetime-local" value={f.due} onChange={(e) => set({ due: e.target.value })} /></Field>
        {initial && (
          <>
            <Field label="Status">
              <select className="select" value={f.status} onChange={(e) => set({ status: e.target.value as CallbackStatus })}>
                {STATUSES.map((s) => <option key={s} value={s}>{s.replace('_', ' ')}</option>)}
              </select>
            </Field>
            <Field label="Assigned to">
              <select className="select" value={f.assigned_to || ''} onChange={(e) => set({ assigned_to: Number(e.target.value) || 0 })}>
                <option value="">Unassigned</option>
                {users.map((u) => <option key={u.id} value={u.id}>{u.display_name || u.username}</option>)}
              </select>
            </Field>
          </>
        )}
        <Field label="Notes" className="span-all"><textarea className="textarea" value={f.notes} onChange={(e) => set({ notes: e.target.value })} /></Field>
      </div>
    </Dialog>
  );
}
