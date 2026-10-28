import { useEffect, useMemo, useState } from 'react';
import { CheckCircle2, Save } from 'lucide-react';
import { get, post, put } from '../../api/client';
import type { QAEvaluation, QAEvaluationDetail, QAScorecard } from '../../api/types_routing';
import { useApi } from '../../lib/hooks';
import { Dialog } from '../../components/ui/Dialog';
import { Field } from '../../components/ui/primitives';
import { toast } from '../../components/ui/Toast';
import { grade, ScoreBadge } from './qaUi';

interface Props { open: boolean; callId: string; evaluationId?: number; onClose: () => void; onSaved: (e: QAEvaluation) => void }

/** Score a call against a scorecard: per-criterion scores and notes, overall comment, draft or final. */
export function EvaluationDialog({ open, callId, evaluationId, onClose, onSaved }: Props) {
  const cards = useApi((s) => get<QAScorecard[]>('/qa/scorecards', undefined, s), [], open);
  const [cardId, setCardId] = useState(0);
  const [scores, setScores] = useState<Record<string, number>>({});
  const [notes, setNotes] = useState<Record<string, string>>({});
  const [comment, setComment] = useState('');
  const [call, setCall] = useState(callId);
  const [busy, setBusy] = useState(false);
  const [finalized, setFinalized] = useState(false);

  useEffect(() => {
    if (!open) return;
    setScores({}); setNotes({}); setComment(''); setCall(callId); setFinalized(false);
    if (!evaluationId) { setCardId(0); return; }
    get<QAEvaluationDetail>(`/qa/evaluations/${evaluationId}`).then(({ evaluation: e }) => {
      setCardId(e.scorecard_id); setScores(e.scores ?? {}); setNotes(e.notes ?? {}); setComment(e.comment); setCall(e.call_id);
      setFinalized(e.status === 'final');
    }).catch(toast.error);
  }, [open, evaluationId, callId]);
  const enabled = (cards.data ?? []).filter((c) => c.enabled || c.id === cardId);
  useEffect(() => { if (open && !cardId && !evaluationId && enabled.length) setCardId(enabled[0].id); }, [open, cardId, evaluationId, enabled]);
  const card = enabled.find((c) => c.id === cardId);
  const g = useMemo(() => (card ? grade(card, scores) : null), [card, scores]);

  const save = async (status: 'draft' | 'final') => {
    if (!card) return;
    setBusy(true);
    try {
      const body = { call_id: call, scorecard_id: card.id, scores, notes, comment, status };
      const saved = evaluationId ? await put<QAEvaluation>(`/qa/evaluations/${evaluationId}`, body) : await post<QAEvaluation>('/qa/evaluations', body);
      toast.success(status === 'final' ? 'Evaluation finalized — the agent can now see it' : 'Draft saved');
      onSaved(saved);
      onClose();
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };

  return (
    <Dialog open={open} onClose={onClose} size="wide" title={evaluationId ? 'Edit evaluation' : 'Evaluate call'}
      footer={<>
        {g && <span className="grow"><ScoreBadge e={{ score: g.score, passed: g.passed, auto_failed: g.autoFailed }} /></span>}
        <button className="btn" onClick={onClose}>Cancel</button>
        {!finalized && <button className="btn" disabled={busy || !card} onClick={() => save('draft')}><Save />Save draft</button>}
        <button className="btn primary" disabled={busy || !card} onClick={() => save('final')}><CheckCircle2 />{finalized ? 'Save final' : 'Finalize'}</button>
      </>}>
      {!enabled.length && !cards.loading ? <div className="alert warn">No scorecards yet — create one in Quality → Scorecards.</div> : (
        <Field label="Scorecard">
          <select className="select" value={cardId || ''} disabled={!!evaluationId} onChange={(e) => { setCardId(Number(e.target.value)); setScores({}); }}>
            {enabled.map((c) => <option key={c.id} value={c.id}>{c.name} (pass ≥ {c.pass_score}%)</option>)}
          </select>
        </Field>
      )}
      {card && (card.criteria ?? []).map((k) => (
        <div key={k.key} className="score-row">
          <div className="row between wrap">
            <span className="strong" dir="auto">{k.label}{k.critical && <span className="badge danger" style={{ marginInlineStart: 6 }}>Critical</span>}</span>
            <span className="small muted">weight {k.weight}</span>
          </div>
          <div className="score-scale" role="radiogroup" aria-label={k.label}>
            {Array.from({ length: k.max_score + 1 }, (_, n) => (
              <button key={n} type="button" role="radio" aria-checked={scores[k.key] === n}
                className={`btn sm ${scores[k.key] === n ? 'active' : ''}`} onClick={() => setScores((s) => ({ ...s, [k.key]: n }))}>{n}</button>
            ))}
          </div>
          <input className="input sm" dir="auto" placeholder="Note (optional)" value={notes[k.key] ?? ''} onChange={(e) => setNotes((x) => ({ ...x, [k.key]: e.target.value }))} />
        </div>
      ))}
      {card && <Field label="Overall comment"><textarea className="textarea" dir="auto" value={comment} onChange={(e) => setComment(e.target.value)} /></Field>}
    </Dialog>
  );
}
