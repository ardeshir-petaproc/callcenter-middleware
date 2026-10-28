import type { QAEvaluation, QAScorecard } from '../../api/types_routing';

/** Same weighted grading as the server (domain.QAEvaluation.Grade), for a live preview. */
export function grade(sc: QAScorecard, scores: Record<string, number>): { score: number; autoFailed: boolean; passed: boolean } {
  let got = 0;
  let total = 0;
  let autoFailed = false;
  for (const c of sc.criteria ?? []) {
    const w = Math.max(c.weight, 1);
    const top = Math.max(c.max_score, 1);
    const s = Math.min(Math.max(scores[c.key] ?? 0, 0), top);
    got += (w * s) / top;
    total += w;
    if (c.critical && s === 0) autoFailed = true;
  }
  let score = total > 0 ? Math.round((got / total) * 1000) / 10 : 0;
  if (autoFailed) score = 0;
  return { score, autoFailed, passed: !autoFailed && score >= sc.pass_score };
}

export function ScoreBadge({ e }: { e: Pick<QAEvaluation, 'score' | 'passed' | 'auto_failed'> }) {
  return (
    <span className="row" style={{ gap: 4 }}>
      <span className={`badge ${e.passed ? 'ok' : 'danger'}`}>{e.score.toFixed(1)}%</span>
      {e.auto_failed ? <span className="badge danger" title="A critical criterion scored 0">Auto-fail</span>
        : <span className="small muted">{e.passed ? 'Pass' : 'Fail'}</span>}
    </span>
  );
}
