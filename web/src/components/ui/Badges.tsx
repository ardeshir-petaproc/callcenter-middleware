import type { AgentStatus, CallState } from '../../api/types';
import { humanize, initials } from '../../lib/format';

const agentLabels: Record<AgentStatus, string> = {
  offline: 'Offline', available: 'Available', ringing: 'Ringing', busy: 'On call', wrapup: 'Wrap-up', paused: 'Paused',
};
const callLabels: Record<CallState, string> = {
  ivr: 'IVR', ai: 'AI agent', queued: 'Queued', ringing: 'Ringing', connected: 'Connected', hold: 'On hold',
  voicemail: 'Voicemail', dialing: 'Dialing', message: 'Message', survey: 'Survey', ended: 'Ended',
};

export function agentStatusLabel(s: AgentStatus | string): string {
  return agentLabels[s as AgentStatus] ?? humanize(s);
}

export function AgentStatusBadge({ status, reason }: { status: AgentStatus; reason?: string }) {
  const live = status === 'ringing';
  return (
    <span className={`badge st st-${status} ${live ? 'pulse' : ''}`}>
      <span className="dot" />
      {agentStatusLabel(status)}{status === 'paused' && reason ? ` · ${reason}` : ''}
    </span>
  );
}

export function CallStateBadge({ state }: { state: CallState }) {
  const live = state === 'ringing' || state === 'queued' || state === 'dialing';
  return (
    <span className={`badge st st-${state} ${live ? 'pulse' : ''}`}>
      <span className="dot" />
      {callLabels[state] ?? humanize(state)}
    </span>
  );
}

const outcomeTone: Record<string, string> = {
  answered: 'ok', ai_resolved: 'violet', abandoned: 'danger', failed: 'danger', no_answer: 'warn',
  voicemail: 'info', callback_requested: 'info', ivr_hangup: '', sent_to_dialplan: '', transferred_external: 'orange',
};

export function OutcomeBadge({ outcome }: { outcome: string }) {
  if (!outcome) return <span className="muted">—</span>;
  return <span className={`badge ${outcomeTone[outcome] ?? ''}`}>{humanize(outcome)}</span>;
}

const sentimentTone: Record<string, string> = { positive: 'ok', neutral: '', negative: 'danger', mixed: 'warn' };
export function SentimentBadge({ sentiment }: { sentiment: string }) {
  if (!sentiment) return null;
  return <span className={`badge ${sentimentTone[sentiment.toLowerCase()] ?? ''}`}>{humanize(sentiment)}</span>;
}

/** Post-call satisfaction score (1-5); renders nothing when unrated. */
export function CsatBadge({ csat }: { csat: number }) {
  if (!csat) return null;
  const tone = csat >= 4 ? 'ok' : csat === 3 ? 'warn' : 'danger';
  return <span className={`badge ${tone}`} title={`Customer rated ${csat} of 5`}>★ {csat}/5</span>;
}

export function Avatar({ name, status }: { name: string; status?: AgentStatus }) {
  return (
    <span className="avatar" aria-hidden>
      {initials(name)}
      {status && <span className={`status-dot st st-${status}`} />}
    </span>
  );
}

export function RoleBadge({ role }: { role: string }) {
  const tone = role === 'admin' ? 'violet' : role === 'supervisor' ? 'info' : '';
  return <span className={`badge ${tone}`}>{humanize(role)}</span>;
}
