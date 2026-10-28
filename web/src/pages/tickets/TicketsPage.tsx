import { useEffect, useMemo, useState } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { Plus, Ticket as TicketIcon } from 'lucide-react';
import { get } from '../../api/client';
import type { Paged } from '../../api/types';
import type { Ticket } from '../../api/types_work';
import { useApi, useDebounced } from '../../lib/hooks';
import { fmtDateTime, humanize } from '../../lib/format';
import { PERMS } from '../../lib/perms';
import { useCan } from '../../state/auth';
import { onLive } from '../../state/ws';
import { useDepartmentNames, useUserNames } from '../../state/lookups';
import { Card, Empty, ErrorBox, PageHeader, Pager, SkeletonRows } from '../../components/ui/primitives';
import { TicketFilters, TICKET_FILTER_KEYS, type TicketFilterValues } from './TicketFilters';
import { PriorityBadge, SlaBadge, TicketStatusBadge } from './ticketUi';
import { NewTicketDialog } from './NewTicketDialog';

const LIMIT = 25;

export default function TicketsPage() {
  const nav = useNavigate();
  const can = useCan();
  const [params, setParams] = useSearchParams();
  const filters = useMemo(() => Object.fromEntries(TICKET_FILTER_KEYS.map((k) => [k, params.get(k) ?? ''])) as unknown as TicketFilterValues, [params]);
  const offset = Number(params.get('offset') ?? 0) || 0;
  const q = useDebounced(filters.q, 350);
  const [creating, setCreating] = useState(false);
  const deptName = useDepartmentNames();
  const userName = useUserNames();
  const query = { ...filters, q, limit: LIMIT, offset };
  const { data, loading, error, reload } = useApi((s) => get<Paged<Ticket>>('/tickets', query, s), [JSON.stringify(query)]);

  // Live refresh when any ticket changes (debounced to one reload per burst).
  useEffect(() => {
    let t: number | undefined;
    const off = onLive('ticket.updated', () => { clearTimeout(t); t = window.setTimeout(reload, 400); });
    return () => { off(); clearTimeout(t); };
  }, [reload]);

  const update = (f: Partial<TicketFilterValues>) => {
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
      <PageHeader title="Tickets" sub={can(PERMS.ticketsAll) ? 'All customer requests' : 'Your tickets and your departments’ tickets'}
        actions={<button className="btn primary" onClick={() => setCreating(true)}><Plus />New ticket</button>} />
      <Card flush>
        <TicketFilters value={filters} onChange={update} />
        {error ? <div style={{ padding: 16 }}><ErrorBox error={error} onRetry={reload} /></div>
          : loading && !data ? <SkeletonRows rows={8} />
          : items.length === 0 ? <Empty title="No tickets found" icon={<TicketIcon />}>Try clearing filters, or create a ticket.</Empty>
          : (
            <div className="table-wrap" style={{ opacity: loading ? 0.6 : 1 }}>
              <table className="table">
                <thead><tr><th>Ticket</th><th>Status</th><th>Priority</th><th className="hide-sm">Department</th><th className="hide-sm">Assignee</th><th>SLA</th><th className="hide-sm">Updated</th></tr></thead>
                <tbody>
                  {items.map((t) => (
                    <tr key={t.id} className="clickable" onClick={() => nav(`/tickets/${t.id}`)}>
                      <td style={{ maxWidth: 380 }}>
                        <div className="strong ellipsis" dir="auto">{t.subject}</div>
                        <div className="sub"><span className="mono">{t.number}</span> · {humanize(t.source)}{t.requester_name || t.requester_phone ? <> · <span dir="auto">{t.requester_name || t.requester_phone}</span></> : null}</div>
                      </td>
                      <td><TicketStatusBadge status={t.status} /></td>
                      <td><PriorityBadge priority={t.priority} /></td>
                      <td className="hide-sm">{deptName(t.department_id)}</td>
                      <td className="hide-sm">{t.assignee_id ? userName(t.assignee_id) : <span className="muted">Unassigned</span>}</td>
                      <td><SlaBadge ticket={t} /></td>
                      <td className="hide-sm tnum">{fmtDateTime(t.updated_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        {data && data.total > 0 && <Pager offset={offset} limit={LIMIT} total={data.total} onChange={setOffset} />}
      </Card>
      <NewTicketDialog open={creating} onClose={() => setCreating(false)} onCreated={(t) => nav(`/tickets/${t.id}`)} />
    </>
  );
}
