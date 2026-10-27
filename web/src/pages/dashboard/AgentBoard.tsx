import { Users } from 'lucide-react';
import type { AgentState } from '../../api/types';
import { useNow, useStore } from '../../lib/store';
import { fmtDuration, fmtLongDuration, secsSince } from '../../lib/format';
import { AgentStatusBadge, Avatar } from '../../components/ui/Badges';
import { Empty } from '../../components/ui/primitives';
import { live } from '../../state/live';
import { callerLabel } from '../../lib/calls';

export function AgentBoard({ agents, hideOffline }: { agents: AgentState[]; hideOffline?: boolean }) {
  const now = useNow();
  const calls = useStore(live, (s) => s.calls);
  const list = hideOffline ? agents.filter((a) => a.status !== 'offline') : agents;
  if (!list.length) return <Empty title="No agents logged in" icon={<Users />}>Agents appear here when they set themselves available.</Empty>;
  return (
    <div className="agent-board">
      {list.map((a) => {
        const call = a.call_id ? calls[a.call_id] : undefined;
        const name = a.display_name || a.username;
        return (
          <div key={a.user_id} className={`agent-card st st-${a.status}`}>
            <div className="row">
              <Avatar name={name} status={a.status} />
              <div className="grow" style={{ minWidth: 0 }}>
                <div className="name ellipsis">{name}</div>
                <div className="small muted">Ext. {a.extension || '—'}</div>
              </div>
              <span className="num strong" title="Time in status">{fmtDuration(secsSince(a.since, now))}</span>
            </div>
            <div className="row between">
              <AgentStatusBadge status={a.status} reason={a.pause_reason} />
              {a.missed_in_row > 0 && <span className="badge warn" title="Missed calls in a row">{a.missed_in_row} missed</span>}
            </div>
            <div className="meta">
              <span className="ellipsis">{call ? `☎ ${callerLabel(call)}` : `${a.calls_today} calls today`}</span>
              <span>{fmtLongDuration(a.talk_secs_today)} talk</span>
            </div>
          </div>
        );
      })}
    </div>
  );
}
