import type { RoutingDecision } from '../../api/types_routing';
import { humanize } from '../../lib/format';
import { useDescribeAction } from '../../components/ActionEditor';

const resultTone: Record<string, string> = { selected: 'ok', unavailable: 'warn', skipped: '' };

/** Explains one routing decision: where the request went, why, the steps taken and the rules considered. */
export function DecisionView({ d }: { d: RoutingDecision }) {
  const describe = useDescribeAction();
  return (
    <div className="col gap-lg">
      <div className="decision-target">
        <div className="small muted">Routed to</div>
        <div className="strong" style={{ fontSize: 17 }} dir="auto">{d.target_name || describe(d.action)}</div>
        <div className="small muted">{describe(d.action)}{d.topic ? <> · topic <strong dir="auto">{d.topic}</strong></> : ' · no topic detected'}</div>
      </div>
      {d.reason && <div><div className="field-label">Reason</div><div dir="auto">{d.reason}</div></div>}
      {(d.steps ?? []).length > 0 && (
        <div>
          <div className="field-label">Steps</div>
          <ol className="steps">{(d.steps ?? []).map((s, i) => <li key={i} dir="auto">{s}</li>)}</ol>
        </div>
      )}
      <div>
        <div className="field-label">Rules considered</div>
        {(d.candidates ?? []).length === 0 ? <div className="small muted">No approved rule matched.</div> : (
          <div className="table-wrap">
            <table className="table">
              <thead><tr><th>Rule</th><th className="num">Score</th><th>Matched</th><th>Result</th></tr></thead>
              <tbody>
                {(d.candidates ?? []).map((c) => (
                  <tr key={c.rule_id}>
                    <td dir="auto">{c.rule_name}</td>
                    <td className="num">{c.score}</td>
                    <td><div className="chips">{(c.matched ?? []).map((m) => <span key={m} className="chip" dir="auto">{m}</span>)}</div></td>
                    <td><span className={`badge ${resultTone[c.result] ?? ''}`}>{humanize(c.result)}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  );
}
