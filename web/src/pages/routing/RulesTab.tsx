import { useMemo, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Archive, Check, GitFork, Pencil, Plus, Trash2, X } from 'lucide-react';
import { post } from '../../api/client';
import type { RoutingRule, RuleStatus } from '../../api/types_routing';
import { useCrud } from '../../lib/crud';
import { PERMS } from '../../lib/perms';
import { fmtDateTime, humanize } from '../../lib/format';
import { useCan } from '../../state/auth';
import { useUserNames } from '../../state/lookups';
import { Card, Empty, ErrorBox, Segmented, SkeletonRows } from '../../components/ui/primitives';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';
import { useDescribeAction } from '../../components/ActionEditor';
import { RevisionsButton } from '../../components/RevisionsButton';
import { RuleEditor } from './RuleEditor';
import { RuleSourceBadge, RuleStatusBadge, useTargetName } from './routingUi';

type Filter = RuleStatus | 'all';
const ORDER: RuleStatus[] = ['draft', 'approved', 'rejected', 'retired'];

export function RulesTab() {
  const can = useCan();
  const canManage = can(PERMS.routingManage);
  const canApprove = can(PERMS.routingApprove);
  const crud = useCrud<RoutingRule>('/routing/rules');
  const [params, setParams] = useSearchParams();
  const filter = (params.get('status') ?? 'all') as Filter;
  const [edit, setEdit] = useState<RoutingRule | null | 'new'>(null);
  const target = useTargetName();
  const describe = useDescribeAction();
  const userName = useUserNames();
  const counts = useMemo(() => Object.fromEntries(ORDER.map((s) => [s, crud.items.filter((r) => r.status === s).length])), [crud.items]);
  const groups = ORDER.filter((s) => filter === 'all' || s === filter)
    .map((s) => ({ status: s, rules: crud.items.filter((r) => r.status === s).sort((a, b) => b.priority - a.priority) }))
    .filter((g) => g.rules.length > 0);

  const review = async (r: RoutingRule, action: 'approve' | 'reject' | 'retire') => {
    if (action !== 'approve' && !(await confirm({ title: `${humanize(action)} rule?`, message: `“${r.name}” will no longer route calls.`, danger: true, confirmLabel: humanize(action) }))) return;
    try {
      const updated = await post<RoutingRule>(`/routing/rules/${r.id}/${action}`, {});
      crud.setData((l) => l?.map((x) => (x.id === r.id ? updated : x)));
      toast.success(action === 'approve' ? 'Rule approved — it now routes calls' : `Rule ${updated.status}`);
    } catch (e) { toast.error(e); }
  };
  const remove = async (r: RoutingRule) => {
    if (!(await confirm({ title: 'Delete rule?', message: `Delete “${r.name}”?`, danger: true, confirmLabel: 'Delete' }))) return;
    try { await crud.remove(r.id); toast.success('Rule deleted'); } catch (e) { toast.error(e); }
  };
  const setFilter = (f: Filter) => { const n = new URLSearchParams(params); if (f === 'all') n.delete('status'); else n.set('status', f); setParams(n, { replace: true }); };

  return (
    <Card flush title={<Segmented<Filter> label="Rule status" value={filter} onChange={setFilter}
      options={[{ value: 'all', label: `All (${crud.items.length})` }, ...ORDER.map((s) => ({ value: s, label: `${humanize(s)} (${counts[s] ?? 0})` }))]} />}
      actions={canManage && <button className="btn sm primary" onClick={() => setEdit('new')}><Plus />New rule</button>}>
      <div className="alert info" style={{ margin: 12 }}><div>Only <strong>approved</strong> rules route calls. New rules and every edit — even of an approved rule — are saved as <strong>drafts</strong> and need approval by someone with routing.approve.</div></div>
      {crud.error ? <div style={{ padding: 16 }}><ErrorBox error={crud.error} onRetry={crud.reload} /></div>
        : crud.loading && !crud.data ? <SkeletonRows rows={5} />
        : groups.length === 0 ? <Empty title="No rules" icon={<GitFork />}>Create a rule, or generate drafts from a job profile.</Empty>
        : groups.map((g) => (
          <div key={g.status}>
            <div className="group-head"><RuleStatusBadge status={g.status} /> <span className="small muted">{g.rules.length} rule(s)</span></div>
            <div className="list">
              {g.rules.map((r) => (
                <div key={r.id} className="list-item" style={{ alignItems: 'flex-start', flexWrap: 'wrap' }}>
                  <div className="grow" style={{ minWidth: 240 }}>
                    <div className="row wrap"><span className="strong" dir="auto">{r.name}</span><RuleSourceBadge source={r.source} /><span className="small muted">priority {r.priority} · v{r.version}</span></div>
                    <div className="small">→ <strong>{target(r)}</strong>{r.fallback?.type ? <span className="muted"> · fallback {describe(r.fallback)}</span> : null}</div>
                    <div className="chips" style={{ marginTop: 4 }}>
                      {(r.topics ?? []).map((t) => <span key={`t${t}`} className="chip" dir="auto">topic: {t}</span>)}
                      {(r.keywords ?? []).map((k) => <span key={`k${k}`} className="chip" dir="auto">“{k}”</span>)}
                      {(r.languages ?? []).map((l) => <span key={`l${l}`} className="chip">lang: {l}</span>)}
                      {r.vip_only && <span className="chip">VIP only</span>}
                    </div>
                    {r.rationale && <div className="small muted" dir="auto" style={{ marginTop: 4 }}>Why: {r.rationale}</div>}
                    {r.status === 'approved' && r.approved_at && <div className="small muted">Approved by {userName(r.approved_by)} {fmtDateTime(r.approved_at)}</div>}
                  </div>
                  <div className="btn-group">
                    {canApprove && r.status === 'draft' && <button className="btn sm success" onClick={() => review(r, 'approve')}><Check />Approve</button>}
                    {canApprove && r.status === 'draft' && <button className="btn sm" onClick={() => review(r, 'reject')}><X />Reject</button>}
                    {canApprove && r.status === 'approved' && <button className="btn sm" onClick={() => review(r, 'retire')}><Archive />Retire</button>}
                    {canManage && <button className="btn sm icon" aria-label="Edit" title="Edit (returns the rule to draft)" onClick={() => setEdit(r)}><Pencil /></button>}
                    <RevisionsButton entity="routing_rule" id={r.id} label="" />
                    {canManage && <button className="btn sm icon" aria-label="Delete" onClick={() => remove(r)}><Trash2 /></button>}
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))}
      <RuleEditor open={edit !== null} rule={edit === 'new' ? null : edit} onClose={() => setEdit(null)}
        onSaved={(r) => crud.setData((l) => (l?.some((x) => x.id === r.id) ? l.map((x) => (x.id === r.id ? r : x)) : [...(l ?? []), r]))} />
    </Card>
  );
}
