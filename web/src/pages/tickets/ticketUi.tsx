import { AlarmClock, Bot, Mail, MessageCircle, PenLine, Phone, type LucideIcon } from 'lucide-react';
import type { Ticket, TicketPriority, TicketStatus } from '../../api/types_work';
import { fmtDateTime, fmtRelative, humanize } from '../../lib/format';
import { useNow } from '../../lib/store';

const statusTone: Record<TicketStatus, string> = {
  new: 'info', assigned: 'violet', in_progress: 'primary', waiting_customer: 'warn', resolved: 'ok', closed: '',
};
const priorityTone: Record<TicketPriority, string> = { low: '', normal: 'info', high: 'orange', urgent: 'danger' };

export const statusLabel = (s: string) => (s === 'waiting_customer' ? 'Waiting on customer' : humanize(s));

export function TicketStatusBadge({ status }: { status: TicketStatus }) {
  return <span className={`badge ${statusTone[status] ?? ''}`}>{statusLabel(status)}</span>;
}

export function PriorityBadge({ priority }: { priority: TicketPriority }) {
  return <span className={`badge ${priorityTone[priority] ?? ''}`}>{humanize(priority)}</span>;
}

export function isOpen(t: Ticket): boolean { return t.status !== 'resolved' && t.status !== 'closed'; }

/** Next SLA deadline of an open ticket: first response (until answered), then resolution. */
export function slaDue(t: Ticket): { label: string; at: string } | null {
  if (!isOpen(t)) return null;
  if (!t.first_response_at && t.response_due_at) {
    const resp = { label: 'Response', at: t.response_due_at };
    if (!t.resolution_due_at || Date.parse(t.response_due_at) <= Date.parse(t.resolution_due_at)) return resp;
  }
  return t.resolution_due_at ? { label: 'Resolution', at: t.resolution_due_at } : null;
}

/** "Due in 2 hours" / "Overdue" badge from the ticket's SLA deadlines. */
export function SlaBadge({ ticket, overdue }: { ticket: Ticket; overdue?: boolean }) {
  const now = useNow();
  const due = slaDue(ticket);
  if (!due) return null;
  const late = overdue || Date.parse(due.at) < now;
  const soon = !late && Date.parse(due.at) - now < 60 * 60 * 1000;
  return (
    <span className={`badge ${late ? 'danger' : soon ? 'warn' : ''}`} title={`${due.label} due ${fmtDateTime(due.at)}`}>
      <AlarmClock size={12} />{late ? `${due.label} overdue` : `${due.label} ${fmtRelative(due.at, now)}`}
    </span>
  );
}

const SOURCES: Record<string, { label: string; icon: LucideIcon }> = {
  call: { label: 'Call', icon: Phone }, email: { label: 'Email', icon: Mail }, message: { label: 'Chat message', icon: MessageCircle },
  manual: { label: 'Manual', icon: PenLine }, ai: { label: 'AI agent', icon: Bot },
};

/** Where a ticket came from (call, email, chat message, manual, AI) with an icon. */
export function SourceLabel({ source }: { source: string }) {
  const s = SOURCES[source] ?? { label: humanize(source), icon: PenLine };
  return <span className="source-label" title={`Source: ${s.label}`}><s.icon size={12} />{s.label}</span>;
}
