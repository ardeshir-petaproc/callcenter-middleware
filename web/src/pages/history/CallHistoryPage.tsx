import { useMemo } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Download, History, PhoneIncoming, PhoneOutgoing, Mic } from 'lucide-react';
import type { Call, Paged } from '../../api/types';
import { authUrl, get } from '../../api/client';
import { useApi, useDebounced } from '../../lib/hooks';
import { dayEndIso, dayStartIso, fmtDateTime, fmtDuration } from '../../lib/format';
import { callerLabel, callerSub } from '../../lib/calls';
import { useQueueNames, useUserNames } from '../../state/lookups';
import { useMe } from '../../state/auth';
import { isSupPlus } from '../../lib/roles';
import { Card, Empty, ErrorBox, PageHeader, Pager, SkeletonRows } from '../../components/ui/primitives';
import { CsatBadge, OutcomeBadge, SentimentBadge } from '../../components/ui/Badges';
import { CallFilters, FILTER_KEYS, type Filters } from './CallFilters';

const LIMIT = 25;

export default function CallHistoryPage() {
  const me = useMe();
  const nav = useNavigate();
  const [params, setParams] = useSearchParams();
  const filters = useMemo(() => Object.fromEntries(FILTER_KEYS.map((k) => [k, params.get(k) ?? ''])) as unknown as Filters, [params]);
  const offset = Number(params.get('offset') ?? 0) || 0;
  const q = useDebounced(filters.q, 350);
  const queueName = useQueueNames();
  const userName = useUserNames();

  const query = useMemo(() => ({
    from: filters.from ? dayStartIso(filters.from) : undefined,
    to: filters.to ? dayEndIso(filters.to) : undefined,
    queue_id: filters.queue_id, agent_id: filters.agent_id, campaign_id: filters.campaign_id, outcome: filters.outcome, direction: filters.direction, q,
  }), [filters.from, filters.to, filters.queue_id, filters.agent_id, filters.campaign_id, filters.outcome, filters.direction, q]);

  const { data, loading, error, reload } = useApi(
    (s) => get<Paged<Call>>('/calls', { ...query, limit: LIMIT, offset }, s), [JSON.stringify(query), offset],
  );

  const update = (f: Partial<Filters>) => {
    const next = new URLSearchParams(params);
    for (const [k, v] of Object.entries(f)) { if (v) next.set(k, v); else next.delete(k); }
    next.delete('offset');
    setParams(next, { replace: true });
  };
  const setOffset = (o: number) => {
    const next = new URLSearchParams(params);
    if (o) next.set('offset', String(o)); else next.delete('offset');
    setParams(next);
  };

  const items = data?.items ?? [];
  return (
    <>
      <PageHeader title="Call history" sub={isSupPlus(me.role) ? 'All calls' : 'Your calls'} actions={
        isSupPlus(me.role) && <a className="btn" href={authUrl('/reports/export.csv', query)} download><Download />Export CSV</a>
      } />
      <Card flush>
        <CallFilters value={filters} onChange={update} showAgent={isSupPlus(me.role)} />
        {error ? <div style={{ padding: 16 }}><ErrorBox error={error} onRetry={reload} /></div>
          : loading && !data ? <SkeletonRows rows={8} />
          : items.length === 0 ? <Empty title="No calls found" icon={<History />}>Try widening the date range or clearing filters.</Empty>
          : (
            <div className="table-wrap" style={{ opacity: loading ? 0.6 : 1 }}>
              <table className="table">
                <thead>
                  <tr>
                    <th>Started</th><th>Caller</th><th className="hide-sm">Queue</th><th>Agent</th><th>Outcome</th>
                    <th className="hide-sm">Disposition</th><th className="num">Wait</th><th className="num">Talk</th><th className="hide-sm" />
                  </tr>
                </thead>
                <tbody>
                  {items.map((c) => (
                    <tr key={c.id} className="clickable" onClick={() => nav(`/calls/${encodeURIComponent(c.id)}`)}>
                      <td className="tnum">{fmtDateTime(c.started_at)}</td>
                      <td>
                        <div className="row" style={{ gap: 6 }}>
                          {c.direction === 'outbound' ? <PhoneOutgoing size={14} className="muted" /> : <PhoneIncoming size={14} className="muted" />}
                          <div className="col" style={{ gap: 0, minWidth: 0 }}>
                            <span className="strong ellipsis">{callerLabel(c)}</span>
                            {callerSub(c) && <span className="sub">{callerSub(c)}</span>}
                          </div>
                        </div>
                      </td>
                      <td className="hide-sm">{queueName(c.queue_id)}</td>
                      <td>{userName(c.agent_id)}</td>
                      <td><OutcomeBadge outcome={c.outcome} /></td>
                      <td className="hide-sm">{c.disposition || <span className="muted">—</span>}</td>
                      <td className="num">{fmtDuration(c.wait_sec)}</td>
                      <td className="num">{fmtDuration(c.talk_sec)}</td>
                      <td className="hide-sm"><div className="row" style={{ gap: 4 }}>{c.recording && <Mic size={14} className="muted" aria-label="Recorded" />}<SentimentBadge sentiment={c.sentiment} /><CsatBadge csat={c.csat} /></div></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        {data && data.total > 0 && <Pager offset={offset} limit={LIMIT} total={data.total} onChange={setOffset} />}
      </Card>
    </>
  );
}
