import { BarChart3 } from 'lucide-react';
import { get } from '../../api/client';
import type { QASummary } from '../../api/types_routing';
import { useApi } from '../../lib/hooks';
import { useUserNames } from '../../state/lookups';
import { Card, Empty, ErrorBox, SkeletonRows } from '../../components/ui/primitives';

/** Final evaluations aggregated per agent. */
export function SummaryTab() {
  const { data, loading, error, reload } = useApi((s) => get<QASummary>('/qa/summary', undefined, s), []);
  const userName = useUserNames();
  const rows = [...(data?.agents ?? [])].sort((a, b) => b.avg_score - a.avg_score);
  return (
    <Card title="Quality by agent" icon={<BarChart3 />} flush>
      {error ? <div style={{ padding: 16 }}><ErrorBox error={error} onRetry={reload} /></div>
        : loading && !data ? <SkeletonRows rows={4} />
        : rows.length === 0 ? <Empty title="No final evaluations yet" />
        : (
          <div className="table-wrap">
            <table className="table">
              <thead><tr><th>Agent</th><th className="num">Evaluations</th><th className="num">Average score</th><th className="num">Passed</th><th className="num">Pass rate</th><th className="num">Auto-failed</th></tr></thead>
              <tbody>
                {rows.map((r) => (
                  <tr key={r.agent_id}>
                    <td>{userName(r.agent_id)}</td>
                    <td className="num">{r.evaluations}</td>
                    <td className="num"><span className="row end" style={{ gap: 6 }}><span className="meter"><span style={{ width: `${Math.min(100, r.avg_score)}%` }} /></span>{r.avg_score.toFixed(1)}%</span></td>
                    <td className="num">{r.passed}</td>
                    <td className="num">{r.evaluations ? Math.round((r.passed / r.evaluations) * 100) : 0}%</td>
                    <td className="num">{r.auto_failed ? <span className="badge danger">{r.auto_failed}</span> : 0}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
    </Card>
  );
}
