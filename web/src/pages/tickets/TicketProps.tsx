import { Fragment, type ReactNode } from 'react';
import { SlidersHorizontal } from 'lucide-react';
import { TICKET_PRIORITIES, TICKET_STATUSES, type Ticket } from '../../api/types_work';
import { fmtDateTime, humanize } from '../../lib/format';
import { useLookup } from '../../state/lookups';
import { Card } from '../../components/ui/primitives';
import { TagInput } from '../../components/ui/TagInput';
import { statusLabel } from './ticketUi';

/** Editable ticket properties; every change is a PATCH recorded in the history. */
export function TicketProps({ ticket: t, onSave }: { ticket: Ticket; onSave: (p: Partial<Ticket>) => void }) {
  const depts = useLookup('departments');
  const users = useLookup('users');
  const rows: [string, ReactNode][] = [
    ['Status', (
      <select className="select sm" value={t.status} onChange={(e) => onSave({ status: e.target.value as Ticket['status'] })} aria-label="Status">
        {TICKET_STATUSES.map((s) => <option key={s} value={s}>{statusLabel(s)}</option>)}
      </select>
    )],
    ['Priority', (
      <select className="select sm" value={t.priority} onChange={(e) => onSave({ priority: e.target.value as Ticket['priority'] })} aria-label="Priority">
        {TICKET_PRIORITIES.map((p) => <option key={p} value={p}>{humanize(p)}</option>)}
      </select>
    )],
    ['Assignee', (
      <select className="select sm" value={t.assignee_id || ''} onChange={(e) => onSave({ assignee_id: Number(e.target.value) || 0 })} aria-label="Assignee">
        <option value="">Unassigned</option>
        {users.map((u) => <option key={u.id} value={u.id}>{u.display_name || u.username}</option>)}
      </select>
    )],
    ['Department', (
      <select className="select sm" value={t.department_id || ''} onChange={(e) => onSave({ department_id: Number(e.target.value) || 0 })} aria-label="Department">
        <option value="">None</option>
        {depts.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
      </select>
    )],
    ['Category', <span dir="auto">{t.category || '—'}</span>],
    ['Requester', <span dir="auto">{[t.requester_name, t.requester_phone, t.requester_email].filter(Boolean).join(' · ') || '—'}</span>],
    ['Response due', t.first_response_at ? `Answered ${fmtDateTime(t.first_response_at)}` : fmtDateTime(t.response_due_at)],
    ['Resolution due', t.resolved_at ? `Resolved ${fmtDateTime(t.resolved_at)}` : fmtDateTime(t.resolution_due_at)],
    ['Updated', fmtDateTime(t.updated_at)],
  ];
  return (
    <Card title="Properties" icon={<SlidersHorizontal />}>
      <dl className="kv" style={{ alignItems: 'center' }}>{rows.map(([k, v]) => <Fragment key={k}><dt>{k}</dt><dd>{v}</dd></Fragment>)}</dl>
      <div className="field" style={{ marginTop: 12 }}>
        <span className="field-label">Tags</span>
        <TagInput value={t.tags ?? []} onChange={(v) => onSave({ tags: v })} />
      </div>
    </Card>
  );
}
