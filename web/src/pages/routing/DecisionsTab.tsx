import { Fragment, useState } from 'react';
import { Link } from 'react-router-dom';
import { ChevronDown, ChevronRight, ScrollText } from 'lucide-react';
import { get } from '../../api/client';
import type { RoutingDecision } from '../../api/types_routing';
import { useApi } from '../../lib/hooks';
import { fmtDateTime } from '../../lib/format';
import { Card, Empty, ErrorBox, SkeletonRows } from '../../components/ui/primitives';
import { DecisionView } from './DecisionView';

/** Log of routing decisions; with callId, only that call's decisions (used on the call detail page). */
export function DecisionsList({ callId, limit = 100 }: { callId?: string; limit?: number }) {
  const { data, loading, error, reload } = useApi((s) => get<RoutingDecision[]>('/routing/decisions', { call_id: callId, limit }, s), [callId, limit]);
  const [open, setOpen] = useState<number | null>(null);
  if (error) return <div style={{ padding: 16 }}><ErrorBox error={error} onRetry={reload} /></div>;
  if (loading && !data) return <SkeletonRows rows={4} />;
  if (!data?.length) return <Empty title="No routing decisions" icon={<ScrollText />}>{callId ? 'This call was not routed by rules.' : 'Decisions appear when calls are routed by the AI or IVR.'}</Empty>;
  return (
    <div className="table-wrap">
      <table className="table">
        <thead><tr><th /><th>When</th>{!callId && <th>Call</th>}<th>Request</th><th>Topic</th><th>Routed to</th></tr></thead>
        <tbody>
          {data.map((d, i) => (
            <Fragment key={d.id ?? i}>
              <tr className="clickable" onClick={() => setOpen(open === i ? null : i)}>
                <td style={{ width: 24 }}>{open === i ? <ChevronDown size={14} /> : <ChevronRight size={14} />}</td>
                <td className="tnum">{fmtDateTime(d.at)}</td>
                {!callId && <td>{d.call_id ? <Link to={`/calls/${encodeURIComponent(d.call_id)}`} onClick={(e) => e.stopPropagation()}>{d.call_id.slice(0, 8)}</Link> : '—'}</td>}
                <td style={{ maxWidth: 320 }}><span className="ellipsis" style={{ display: 'block' }} dir="auto">{d.request?.text || '—'}</span></td>
                <td dir="auto">{d.topic || '—'}</td>
                <td dir="auto">{d.target_name || '—'}</td>
              </tr>
              {open === i && <tr><td colSpan={callId ? 5 : 6} style={{ background: 'var(--surface-2)' }}><DecisionView d={d} /></td></tr>}
            </Fragment>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function DecisionsTab() {
  return <Card title="Decision log" icon={<ScrollText />} flush><DecisionsList /></Card>;
}
