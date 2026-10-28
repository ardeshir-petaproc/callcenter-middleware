import { useState } from 'react';
import { Link } from 'react-router-dom';
import { CheckCheck, Star } from 'lucide-react';
import { get, post } from '../../api/client';
import type { QAEvaluation, QAScorecard } from '../../api/types_routing';
import { useApi } from '../../lib/hooks';
import { fmtDateTime } from '../../lib/format';
import { useMe } from '../../state/auth';
import { useUserNames } from '../../state/lookups';
import { Card, Empty, ErrorBox, PageHeader, SkeletonRows } from '../../components/ui/primitives';
import { toast } from '../../components/ui/Toast';
import { ScoreBadge } from './qaUi';

/** The signed-in agent's final evaluations, with a breakdown and an acknowledge step. */
export default function MyEvaluationsPage() {
  const me = useMe();
  const list = useApi((s) => get<QAEvaluation[]>('/qa/evaluations', { agent_id: me.id }, s), [me.id]);
  const cards = useApi((s) => get<QAScorecard[]>('/qa/scorecards', undefined, s), []);
  const items = (list.data ?? []).filter((e) => e.status === 'final' && e.agent_id === me.id);
  return (
    <>
      <PageHeader title="My evaluations" sub="Quality reviews of your calls" />
      {list.error ? <ErrorBox error={list.error} onRetry={list.reload} />
        : list.loading && !list.data ? <div className="card"><SkeletonRows rows={4} /></div>
        : items.length === 0 ? <div className="card"><Empty title="No evaluations yet" icon={<Star />}>Finalized reviews of your calls will appear here.</Empty></div>
        : (
          <div className="stack">
            {items.map((e) => (
              <EvaluationCard key={e.id} e={e} card={cards.data?.find((c) => c.id === e.scorecard_id)}
                onAck={(u) => list.setData((l) => l?.map((x) => (x.id === u.id ? u : x)))} />
            ))}
          </div>
        )}
    </>
  );
}

function EvaluationCard({ e, card, onAck }: { e: QAEvaluation; card?: QAScorecard; onAck: (e: QAEvaluation) => void }) {
  const userName = useUserNames();
  const [comment, setComment] = useState('');
  const [busy, setBusy] = useState(false);
  const ack = async () => {
    setBusy(true);
    try { onAck(await post<QAEvaluation>(`/qa/evaluations/${e.id}/acknowledge`, { comment })); toast.success('Acknowledged'); } catch (err) { toast.error(err); } finally { setBusy(false); }
  };
  return (
    <Card title={<span className="row wrap"><span dir="auto">{card?.name ?? 'Evaluation'}</span><ScoreBadge e={e} /></span>}
      actions={<span className="small muted">Call <Link to={`/calls/${encodeURIComponent(e.call_id)}`}>{e.call_id.slice(0, 8)}</Link> · {fmtDateTime(e.updated_at)} · by {userName(e.evaluator_id)}</span>}>
      <div className="col gap-lg">
        {card && (
          <table className="table">
            <thead><tr><th>Criterion</th><th className="num">Score</th><th>Note</th></tr></thead>
            <tbody>
              {(card.criteria ?? []).map((k) => (
                <tr key={k.key}>
                  <td dir="auto">{k.label}{k.critical && <span className="badge danger" style={{ marginInlineStart: 6 }}>Critical</span>}</td>
                  <td className="num">{e.scores?.[k.key] ?? 0} / {k.max_score}</td>
                  <td dir="auto" className="small">{e.notes?.[k.key] || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
        {e.comment && <div><div className="field-label">Reviewer comment</div><p className="pre" dir="auto">{e.comment}</p></div>}
        {e.acknowledged ? (
          <div className="alert info"><CheckCheck /><div>You acknowledged this evaluation{e.agent_comment ? <>: <span dir="auto">“{e.agent_comment}”</span></> : '.'}</div></div>
        ) : (
          <div className="row wrap">
            <input className="input grow" dir="auto" placeholder="Your comment (optional)" value={comment} onChange={(ev) => setComment(ev.target.value)} />
            <button className="btn primary" disabled={busy} onClick={ack}><CheckCheck />Acknowledge</button>
          </div>
        )}
      </div>
    </Card>
  );
}
