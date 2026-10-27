import { Search, X } from 'lucide-react';
import { OUTCOMES } from '../../api/types';
import { useLookup } from '../../state/lookups';
import { humanize } from '../../lib/format';

export interface Filters { from: string; to: string; queue_id: string; agent_id: string; campaign_id: string; outcome: string; direction: string; q: string }
export const FILTER_KEYS: (keyof Filters)[] = ['from', 'to', 'queue_id', 'agent_id', 'campaign_id', 'outcome', 'direction', 'q'];

export function CallFilters({ value, onChange, showAgent }: { value: Filters; onChange: (f: Partial<Filters>) => void; showAgent: boolean }) {
  const queues = useLookup('queues');
  const users = useLookup('users');
  const campaigns = useLookup('campaigns');
  const active = FILTER_KEYS.some((k) => value[k]);
  return (
    <div className="row wrap" style={{ gap: 8, padding: '12px 16px', borderBottom: '1px solid var(--border)' }}>
      <div className="input-icon" style={{ flex: '1 1 200px', maxWidth: 280 }}>
        <Search />
        <input className="input sm" placeholder="Search number, name, notes…" value={value.q} onChange={(e) => onChange({ q: e.target.value })} aria-label="Search" />
      </div>
      <input className="input sm" type="date" style={{ width: 140 }} value={value.from} onChange={(e) => onChange({ from: e.target.value })} aria-label="From date" />
      <span className="muted small">to</span>
      <input className="input sm" type="date" style={{ width: 140 }} value={value.to} onChange={(e) => onChange({ to: e.target.value })} aria-label="To date" />
      <select className="select sm" style={{ width: 130 }} value={value.queue_id} onChange={(e) => onChange({ queue_id: e.target.value })} aria-label="Queue">
        <option value="">All queues</option>
        {queues.map((q) => <option key={q.id} value={q.id}>{q.name}</option>)}
      </select>
      {showAgent && (
        <select className="select sm" style={{ width: 140 }} value={value.agent_id} onChange={(e) => onChange({ agent_id: e.target.value })} aria-label="Agent">
          <option value="">All agents</option>
          {users.map((u) => <option key={u.id} value={u.id}>{u.display_name || u.username}</option>)}
        </select>
      )}
      {campaigns.length > 0 && (
        <select className="select sm" style={{ width: 150 }} value={value.campaign_id} onChange={(e) => onChange({ campaign_id: e.target.value })} aria-label="Campaign">
          <option value="">All campaigns</option>
          {campaigns.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
        </select>
      )}
      <select className="select sm" style={{ width: 150 }} value={value.outcome} onChange={(e) => onChange({ outcome: e.target.value })} aria-label="Outcome">
        <option value="">All outcomes</option>
        {OUTCOMES.map((o) => <option key={o} value={o}>{humanize(o)}</option>)}
      </select>
      <select className="select sm" style={{ width: 120 }} value={value.direction} onChange={(e) => onChange({ direction: e.target.value })} aria-label="Direction">
        <option value="">Both ways</option>
        <option value="inbound">Inbound</option>
        <option value="outbound">Outbound</option>
      </select>
      {active && (
        <button className="btn ghost sm" onClick={() => onChange(Object.fromEntries(FILTER_KEYS.map((k) => [k, ''])) as unknown as Filters)}>
          <X />Clear
        </button>
      )}
    </div>
  );
}
