import { useState } from 'react';
import { Link } from 'react-router-dom';
import { ClipboardCheck } from 'lucide-react';
import { get } from '../../api/client';
import type { QAEvaluation, QAScorecard } from '../../api/types_routing';
import { useApi } from '../../lib/hooks';
import { fmtDateTime } from '../../lib/format';
import { useLookup, useUserNames } from '../../state/lookups';
import { Card, Empty, ErrorBox, SkeletonRows } from '../../components/ui/primitives';
import { ScoreBadge } from './qaUi';
import { EvaluationDialog } from './EvaluationDialog';

/** Evaluations table; agentId/callId narrow it (call detail page uses callId). */
export function EvaluationsTable({ agentId, callId, onEdit, nonce = 0 }: { agentId?: number; callId?: string; onEdit?: (e: QAEvaluation) => void; nonce?: number }) {
  const { data, loading, error, reload } = useApi((s) => get<QAEvaluation[]>('/qa/evaluations', { agent_id: agentId, call_id: callId }, s), [agentId, callId, nonce]);
  const cards = useApi((s) => get<QAScorecard[]>('/qa/scorecards', undefined, s), []);
  const userName = useUserNames();
  const cardName = (id: number) => cards.data?.find((c) => c.id === id)?.name ?? `#${id}`;
  if (error) return <div style={{ padding: 16 }}><ErrorBox error={error} onRetry={reload} /></div>;
  if (loading && !data) return <SkeletonRows rows={4} />;
  if (!data?.length) return <Empty title="No evaluations yet" icon={<ClipboardCheck />} />;
  return (
    <div className="table-wrap">
      <table className="table">
        <thead><tr><th>Date</th>{!callId && <th>Call</th>}<th>Agent</th><th>Scorecard</th><th>Score</th><th>Status</th><th className="hide-sm">Evaluator</th></tr></thead>
        <tbody>
          {data.map((e) => (
            <tr key={e.id} className={onEdit ? 'clickable' : undefined} onClick={() => onEdit?.(e)}>
              <td className="tnum">{fmtDateTime(e.updated_at || e.created_at)}</td>
              {!callId && <td><Link to={`/calls/${encodeURIComponent(e.call_id)}`} onClick={(ev) => ev.stopPropagation()}>{e.call_id.slice(0, 8)}</Link></td>}
              <td>{userName(e.agent_id)}</td>
              <td dir="auto">{cardName(e.scorecard_id)}</td>
              <td><ScoreBadge e={e} /></td>
              <td><span className={`badge ${e.status === 'final' ? 'ok' : 'warn'}`}>{e.status === 'final' ? 'Final' : 'Draft'}</span>{e.acknowledged && <span className="badge info" style={{ marginInlineStart: 4 }} title={e.agent_comment}>Acknowledged</span>}</td>
              <td className="hide-sm">{userName(e.evaluator_id)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function EvaluationsTab() {
  const users = useLookup('users');
  const [agent, setAgent] = useState(0);
  const [edit, setEdit] = useState<QAEvaluation | null>(null);
  const [nonce, setNonce] = useState(0);
  return (
    <Card title="Evaluations" icon={<ClipboardCheck />} flush actions={
      <select className="select sm" style={{ width: 180 }} value={agent || ''} onChange={(e) => setAgent(Number(e.target.value) || 0)} aria-label="Agent">
        <option value="">All agents</option>
        {users.map((u) => <option key={u.id} value={u.id}>{u.display_name || u.username}</option>)}
      </select>
    }>
      <div className="small muted" style={{ padding: '10px 16px 0' }}>Evaluate a call from its detail page (Call history → call → Evaluate). Click a row to edit.</div>
      <EvaluationsTable agentId={agent || undefined} onEdit={setEdit} nonce={nonce} />
      <EvaluationDialog open={!!edit} callId={edit?.call_id ?? ''} evaluationId={edit?.id} onClose={() => setEdit(null)} onSaved={() => setNonce((n) => n + 1)} />
    </Card>
  );
}
