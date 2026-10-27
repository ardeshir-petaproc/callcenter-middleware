import { PhoneIncoming, PhoneOutgoing, Radio, Eye } from 'lucide-react';
import type { CallView } from '../../api/types';
import { useNow } from '../../lib/store';
import { callerLabel, callerSub, liveDuration, liveWait } from '../../lib/calls';
import { fmtDuration } from '../../lib/format';
import { useQueueNames, useUserNames } from '../../state/lookups';
import { CallStateBadge } from '../ui/Badges';
import { Empty } from '../ui/primitives';
import { CallActions } from './CallActions';

interface Props {
  calls: CallView[];
  actions?: boolean;
  onOpen?: (id: string) => void;
  limit?: number;
}

export function LiveCallsTable({ calls, actions = true, onOpen, limit }: Props) {
  const now = useNow();
  const queueName = useQueueNames();
  const userName = useUserNames();
  const rows = limit ? calls.slice(0, limit) : calls;
  if (!calls.length) return <Empty title="No active calls" icon={<Radio />}>Live calls appear here the moment they arrive.</Empty>;
  return (
    <div className="table-wrap">
      <table className="table">
        <thead>
          <tr>
            <th>State</th>
            <th>Caller</th>
            <th className="hide-sm">Queue</th>
            <th>Agent</th>
            <th className="hide-sm">IVR path</th>
            <th className="num">Wait</th>
            <th className="num">Duration</th>
            {actions && <th className="right">Actions</th>}
          </tr>
        </thead>
        <tbody>
          {rows.map((c) => {
            const wait = liveWait(c, now);
            const ringingNames = c.state === 'ringing' && c.ringing_agents?.length ? c.ringing_agents.map(userName).join(', ') : '';
            return (
              <tr key={c.id} className={onOpen ? 'clickable' : ''} onClick={() => onOpen?.(c.id)}>
                <td>
                  <div className="row" style={{ gap: 6 }}>
                    <CallStateBadge state={c.state} />
                    {c.monitors > 0 && <span className="badge violet" title={`${c.monitors} supervisor(s) monitoring`}><Eye size={12} />{c.monitors}</span>}
                  </div>
                </td>
                <td>
                  <div className="row" style={{ gap: 6 }}>
                    {c.direction === 'outbound' ? <PhoneOutgoing size={14} className="muted" aria-label="Outbound" /> : <PhoneIncoming size={14} className="muted" aria-label="Inbound" />}
                    <div className="col" style={{ gap: 0, minWidth: 0 }}>
                      <span className="strong ellipsis">{callerLabel(c)}</span>
                      {callerSub(c) && <span className="sub">{callerSub(c)}</span>}
                    </div>
                  </div>
                </td>
                <td className="hide-sm">{c.queue_id ? queueName(c.queue_id) : <span className="muted">—</span>}</td>
                <td>{c.agent_id ? userName(c.agent_id) : ringingNames ? <span className="sub">→ {ringingNames}</span> : <span className="muted">—</span>}</td>
                <td className="hide-sm">{c.ivr_path ? <code>{c.ivr_path}</code> : <span className="muted">—</span>}</td>
                <td className={`num ${c.state === 'queued' && wait > 60 ? 'strong' : ''}`} style={c.state === 'queued' && wait > 60 ? { color: 'var(--danger)' } : undefined}>
                  {wait ? fmtDuration(wait) : '—'}
                </td>
                <td className="num">{fmtDuration(liveDuration(c, now))}</td>
                {actions && (
                  <td className="actions">
                    <CallActions call={c} compact onTranscript={onOpen ? () => onOpen(c.id) : undefined} />
                  </td>
                )}
              </tr>
            );
          })}
        </tbody>
      </table>
      {limit && calls.length > limit && <div className="pager">+{calls.length - limit} more active calls</div>}
    </div>
  );
}
