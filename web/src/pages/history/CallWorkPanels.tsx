import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { ClipboardCheck, GitFork, Ticket as TicketIcon, TicketPlus } from 'lucide-react';
import { get, post } from '../../api/client';
import type { Call, Paged } from '../../api/types';
import type { Ticket } from '../../api/types_work';
import { useApi } from '../../lib/hooks';
import { PERMS } from '../../lib/perms';
import { useCan } from '../../state/auth';
import { Card, Empty } from '../../components/ui/primitives';
import { toast } from '../../components/ui/Toast';
import { PriorityBadge, TicketStatusBadge } from '../tickets/ticketUi';
import { DecisionsList } from '../routing/DecisionsTab';
import { EvaluationsTable } from '../quality/EvaluationsTab';
import { EvaluationDialog } from '../quality/EvaluationDialog';

/** Opens a ticket prefilled from the call (summary and transcript excerpt) and navigates to it. */
export function CreateTicketButton({ callId }: { callId: string }) {
  const nav = useNavigate();
  const [busy, setBusy] = useState(false);
  const create = async () => {
    setBusy(true);
    try {
      const t = await post<Ticket>(`/calls/${encodeURIComponent(callId)}/ticket`, {});
      toast.success(`Ticket ${t.number} created`);
      nav(`/tickets/${t.id}`);
    } catch (e) { toast.error(e); setBusy(false); }
  };
  return <button className="btn" disabled={busy} onClick={create}><TicketPlus />Create ticket</button>;
}

export function CallTickets({ callId }: { callId: string }) {
  const { data } = useApi((s) => get<Paged<Ticket>>('/tickets', { call_id: callId, limit: 20 }, s), [callId]);
  const items = data?.items ?? [];
  if (!items.length) return null;
  return (
    <Card title="Tickets" icon={<TicketIcon />} flush>
      <div className="list">
        {items.map((t) => (
          <Link key={t.id} to={`/tickets/${t.id}`} className="list-item clickable">
            <div className="grow"><div className="strong ellipsis" dir="auto">{t.subject}</div><div className="small muted mono">{t.number}</div></div>
            <PriorityBadge priority={t.priority} /><TicketStatusBadge status={t.status} />
          </Link>
        ))}
      </div>
    </Card>
  );
}

export function CallRouting({ callId }: { callId: string }) {
  const can = useCan();
  if (!can(PERMS.routingManage) && !can(PERMS.reportsView)) return null;
  return <Card title="Routing decisions" icon={<GitFork />} flush><DecisionsList callId={callId} limit={20} /></Card>;
}

/** Evaluations of this call; reviewers can open the evaluation form. */
export function CallQuality({ call }: { call: Call }) {
  const can = useCan();
  const reviewer = can(PERMS.qaEvaluate);
  const [open, setOpen] = useState<{ id?: number } | null>(null);
  const [nonce, setNonce] = useState(0);
  const evaluable = !!call.agent_id && !!call.answered_at && !!call.ended_at;
  return (
    <Card title="Quality" icon={<ClipboardCheck />} flush actions={reviewer && (
      <button className="btn sm primary" disabled={!evaluable} title={evaluable ? undefined : 'Only finished calls answered by an agent can be evaluated'} onClick={() => setOpen({})}>
        <ClipboardCheck />Evaluate
      </button>
    )}>
      {reviewer || call.agent_id ? <EvaluationsTable callId={call.id} nonce={nonce} onEdit={reviewer ? (e) => setOpen({ id: e.id }) : undefined} /> : <Empty title="No evaluations" />}
      {reviewer && <EvaluationDialog open={!!open} callId={call.id} evaluationId={open?.id} onClose={() => setOpen(null)} onSaved={() => setNonce((n) => n + 1)} />}
    </Card>
  );
}
