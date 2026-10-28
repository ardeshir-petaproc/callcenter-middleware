import { Search, X } from 'lucide-react';
import { TICKET_PRIORITIES, TICKET_STATUSES } from '../../api/types_work';
import { humanize } from '../../lib/format';
import { useLookup } from '../../state/lookups';
import { statusLabel } from './ticketUi';

export interface TicketFilterValues { status: string; priority: string; department_id: string; assignee_id: string; mine: string; overdue: string; q: string }
export const TICKET_FILTER_KEYS: (keyof TicketFilterValues)[] = ['status', 'priority', 'department_id', 'assignee_id', 'mine', 'overdue', 'q'];

export function TicketFilters({ value, onChange }: { value: TicketFilterValues; onChange: (f: Partial<TicketFilterValues>) => void }) {
  const depts = useLookup('departments');
  const users = useLookup('users');
  const active = TICKET_FILTER_KEYS.some((k) => value[k]);
  return (
    <div className="row wrap" style={{ gap: 8, padding: '12px 16px', borderBottom: '1px solid var(--border)' }}>
      <div className="input-icon" style={{ flex: '1 1 200px', maxWidth: 280 }}>
        <Search />
        <input className="input sm" dir="auto" placeholder="Search subject, number, requester…" value={value.q} onChange={(e) => onChange({ q: e.target.value })} aria-label="Search tickets" />
      </div>
      <select className="select sm" style={{ width: 160 }} value={value.status} onChange={(e) => onChange({ status: e.target.value })} aria-label="Status">
        <option value="">All statuses</option>
        {TICKET_STATUSES.map((s) => <option key={s} value={s}>{statusLabel(s)}</option>)}
      </select>
      <select className="select sm" style={{ width: 130 }} value={value.priority} onChange={(e) => onChange({ priority: e.target.value })} aria-label="Priority">
        <option value="">All priorities</option>
        {TICKET_PRIORITIES.map((p) => <option key={p} value={p}>{humanize(p)}</option>)}
      </select>
      <select className="select sm" style={{ width: 150 }} value={value.department_id} onChange={(e) => onChange({ department_id: e.target.value })} aria-label="Department">
        <option value="">All departments</option>
        {depts.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
      </select>
      {!value.mine && (
        <select className="select sm" style={{ width: 150 }} value={value.assignee_id} onChange={(e) => onChange({ assignee_id: e.target.value })} aria-label="Assignee">
          <option value="">Any assignee</option>
          {users.map((u) => <option key={u.id} value={u.id}>{u.display_name || u.username}</option>)}
        </select>
      )}
      <label className="checkbox small"><input type="checkbox" checked={!!value.mine} onChange={(e) => onChange({ mine: e.target.checked ? '1' : '', assignee_id: '' })} />Assigned to me</label>
      <label className="checkbox small"><input type="checkbox" checked={!!value.overdue} onChange={(e) => onChange({ overdue: e.target.checked ? '1' : '' })} />Overdue</label>
      {active && (
        <button className="btn ghost sm" onClick={() => onChange(Object.fromEntries(TICKET_FILTER_KEYS.map((k) => [k, ''])) as unknown as TicketFilterValues)}>
          <X />Clear
        </button>
      )}
    </div>
  );
}
