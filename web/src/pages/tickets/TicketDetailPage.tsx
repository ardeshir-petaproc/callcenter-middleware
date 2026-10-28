import { useEffect } from 'react';
import { Link, useParams } from 'react-router-dom';
import { ArrowLeft, FileText } from 'lucide-react';
import { get, patch } from '../../api/client';
import type { Ticket, TicketDetail } from '../../api/types_work';
import { useApi } from '../../lib/hooks';
import { fmtDateTime, humanize } from '../../lib/format';
import { onLive } from '../../state/ws';
import { Card, ErrorBox, PageHeader, SkeletonRows } from '../../components/ui/primitives';
import { toast } from '../../components/ui/Toast';
import { PriorityBadge, SlaBadge, TicketStatusBadge } from './ticketUi';
import { TicketProps } from './TicketProps';
import { TicketThread } from './TicketThread';
import { TicketCalls, TicketHistory } from './TicketActivity';

export default function TicketDetailPage() {
  const { id = '' } = useParams();
  const { data, loading, error, reload, setData } = useApi((s) => get<TicketDetail>(`/tickets/${id}`, undefined, s), [id]);

  useEffect(() => onLive('ticket.updated', (d) => {
    const t = d as Partial<Ticket> | null;
    if (!t || t.id === undefined || String(t.id) === id) reload();
  }), [id, reload]);

  if (error) return <ErrorBox error={error} onRetry={reload} />;
  if (loading && !data) return <div className="card"><SkeletonRows rows={8} /></div>;
  if (!data) return null;
  const t = data.ticket;

  const save = async (p: Partial<Ticket>) => {
    try {
      const updated = await patch<Ticket>(`/tickets/${t.id}`, p);
      setData((d) => (d ? { ...d, ticket: updated } : d));
      reload(); // refresh history
    } catch (e) { toast.error(e); }
  };

  return (
    <>
      <Link to="/tickets" className="row small" style={{ gap: 4 }}><ArrowLeft size={14} />Tickets</Link>
      <PageHeader
        title={<span className="row wrap"><span className="mono muted">{t.number}</span><span dir="auto">{t.subject}</span></span>}
        sub={<span className="row wrap"><TicketStatusBadge status={t.status} /><PriorityBadge priority={t.priority} /><SlaBadge ticket={t} overdue={data.overdue} />
          <span>{humanize(t.source)} · opened {fmtDateTime(t.created_at)}</span></span>}
      />
      <div className="ticket-grid">
        <div className="stack">
          <Card title="Description" icon={<FileText />}>
            {t.description ? <p className="pre" dir="auto">{t.description}</p> : <span className="small muted">No description.</span>}
          </Card>
          <TicketThread ticketId={t.id} comments={data.comments ?? []} onPosted={reload} />
        </div>
        <div className="stack">
          <TicketProps ticket={t} onSave={save} />
          <TicketCalls calls={data.calls ?? []} />
          <TicketHistory events={data.history ?? []} />
        </div>
      </div>
    </>
  );
}
