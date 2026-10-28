import { Bot, User } from 'lucide-react';
import type { RoutingRule, RuleStatus, TargetType } from '../../api/types_routing';
import { humanize } from '../../lib/format';
import { useDepartmentNames, useLookup, useQueueNames, useUserNames } from '../../state/lookups';

const tone: Record<RuleStatus, string> = { draft: 'warn', approved: 'ok', rejected: 'danger', retired: '' };
export function RuleStatusBadge({ status }: { status: RuleStatus }) {
  return <span className={`badge ${tone[status] ?? ''}`}>{humanize(status)}</span>;
}

export function RuleSourceBadge({ source }: { source: string }) {
  return source === 'ai'
    ? <span className="badge violet" title="Drafted by AI from a job profile"><Bot size={11} />AI</span>
    : <span className="badge" title="Written by a person"><User size={11} />Manual</span>;
}

export const TARGET_TYPES: { value: TargetType; label: string }[] = [
  { value: 'job_profile', label: 'Job profile' }, { value: 'user', label: 'Employee' },
  { value: 'department', label: 'Department' }, { value: 'queue', label: 'Queue' },
];

/** "Job profile: Billing specialist" style label for a rule target. */
export function useTargetName(): (r: Pick<RoutingRule, 'target_type' | 'target_id'>) => string {
  const users = useUserNames();
  const queues = useQueueNames();
  const depts = useDepartmentNames();
  const jobs = useLookup('jobProfiles');
  return (r) => {
    switch (r.target_type) {
      case 'user': return `Employee: ${users(r.target_id)}`;
      case 'queue': return `Queue: ${queues(r.target_id)}`;
      case 'department': return `Department: ${depts(r.target_id)}`;
      case 'job_profile': return `Job profile: ${jobs.find((j) => j.id === r.target_id)?.title ?? `#${r.target_id}`}`;
      default: return humanize(r.target_type);
    }
  };
}

/** Select for the target of a rule matching its target type. */
export function TargetSelect({ type, value, onChange, disabled }: { type: TargetType; value: number; onChange: (id: number) => void; disabled?: boolean }) {
  const users = useLookup('users');
  const queues = useLookup('queues');
  const depts = useLookup('departments');
  const jobs = useLookup('jobProfiles');
  const opts = type === 'user' ? users.map((u) => ({ id: u.id, name: u.display_name || u.username }))
    : type === 'queue' ? queues.map((q) => ({ id: q.id, name: q.name }))
    : type === 'department' ? depts.map((d) => ({ id: d.id, name: d.name }))
    : jobs.map((j) => ({ id: j.id, name: j.title }));
  return (
    <select className="select" value={value || ''} disabled={disabled} onChange={(e) => onChange(Number(e.target.value) || 0)} aria-label="Target">
      <option value="">Select…</option>
      {opts.map((o) => <option key={o.id} value={o.id}>{o.name}</option>)}
    </select>
  );
}
