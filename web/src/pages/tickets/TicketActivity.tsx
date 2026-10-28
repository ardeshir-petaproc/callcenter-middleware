import { Link } from 'react-router-dom';
import { History, PhoneCall } from 'lucide-react';
import type { Call } from '../../api/types';
import type { TicketEvent } from '../../api/types_work';
import { fmtDateTime, fmtDuration, humanize } from '../../lib/format';
import { callerLabel } from '../../lib/calls';
import { useDepartmentNames, useUserNames } from '../../state/lookups';
import { Card, Empty } from '../../components/ui/primitives';
import { OutcomeBadge } from '../../components/ui/Badges';
import { statusLabel } from './ticketUi';

export function TicketCalls({ calls }: { calls: Call[] }) {
  if (!calls.length) return null;
  return (
    <Card title="Linked calls" icon={<PhoneCall />} flush>
      <div className="list">
        {calls.map((c) => (
          <Link key={c.id} to={`/calls/${encodeURIComponent(c.id)}`} className="list-item clickable">
            <div className="grow"><div className="strong" dir="auto">{callerLabel(c)}</div><div className="small muted">{fmtDateTime(c.started_at)} · {fmtDuration(c.talk_sec)}</div></div>
            <OutcomeBadge outcome={c.outcome} />
          </Link>
        ))}
      </div>
    </Card>
  );
}

export function TicketHistory({ events }: { events: TicketEvent[] }) {
  const userName = useUserNames();
  const deptName = useDepartmentNames();
  const fmt = (type: string, v: string) => {
    if (!v) return '—';
    if (type === 'status') return statusLabel(v);
    if (type === 'assignee' || type === 'assignee_id') return v === '0' ? 'Unassigned' : userName(Number(v));
    if (type === 'department' || type === 'department_id') return v === '0' ? 'None' : deptName(Number(v));
    return v;
  };
  return (
    <Card title="History" icon={<History />}>
      {events.length === 0 ? <Empty title="No changes yet" /> : (
        <ul className="timeline">
          {[...events].reverse().map((e) => (
            <li key={e.id}>
              <span className="t-type">{humanize(e.type)}</span><span className="t-time">{fmtDateTime(e.at)}</span>
              <div className="t-detail">
                {(e.from || e.to) && <><span dir="auto">{fmt(e.type, e.from)}</span> → <span dir="auto">{fmt(e.type, e.to)}</span> · </>}
                <span dir="auto">{e.actor_name || 'System'}</span>
              </div>
            </li>
          ))}
        </ul>
      )}
    </Card>
  );
}
