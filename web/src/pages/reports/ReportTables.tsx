import type { AgentTime, GroupStat } from '../../api/types';
import { fmtCsat, fmtDuration, fmtLongDuration, fmtNum, fmtPct, pctValue } from '../../lib/format';
import { Empty } from '../../components/ui/primitives';

function SlaCell({ s }: { s: GroupStat }) {
  if (!s.answered) return <td className="num muted">—</td>;
  const v = pctValue(s.sla_percent);
  const color = v >= 80 ? 'var(--ok)' : v >= 60 ? 'var(--warn)' : 'var(--danger)';
  return <td className="num strong" style={{ color }}>{fmtPct(s.sla_percent)}</td>;
}

export function QueueTable({ rows, name }: { rows: GroupStat[]; name: (id: number) => string }) {
  if (!rows.length) return <Empty title="No queue activity" />;
  return (
    <div className="table-wrap">
      <table className="table">
        <thead><tr><th>Queue</th><th className="num">Calls</th><th className="num">Answered</th><th className="num">Abandoned</th><th className="num">Abandon %</th><th className="num">SLA</th><th className="num">Avg wait</th><th className="num">Max wait</th><th className="num">Avg talk</th><th className="num">Callbacks</th><th className="num">Voicemail</th><th className="num">CSAT</th></tr></thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r.id}>
              <td className="strong">{name(r.id)}</td>
              <td className="num">{fmtNum(r.total)}</td><td className="num">{fmtNum(r.answered)}</td><td className="num">{fmtNum(r.abandoned)}</td>
              <td className="num">{r.total ? fmtPct(r.abandon_rate) : '—'}</td><SlaCell s={r} />
              <td className="num">{fmtDuration(r.avg_wait_sec)}</td><td className="num">{fmtDuration(r.max_wait_sec)}</td><td className="num">{fmtDuration(r.avg_talk_sec)}</td>
              <td className="num">{fmtNum(r.callbacks)}</td><td className="num">{fmtNum(r.voicemail)}</td>
              <td className="num" title={r.csat_count ? `${r.csat_count} ratings` : undefined}>{fmtCsat(r.csat_avg, r.csat_count)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

const TIME_KEYS = ['available', 'busy', 'ringing', 'wrapup', 'paused'];

export function AgentTable({ stats, times, name }: { stats: GroupStat[]; times: AgentTime[]; name: (id: number) => string }) {
  const ids = Array.from(new Set([...stats.map((s) => s.id), ...times.map((t) => t.user_id)])).filter(Boolean);
  if (!ids.length) return <Empty title="No agent activity" />;
  const rows = ids.map((id) => ({ id, s: stats.find((x) => x.id === id), t: times.find((x) => x.user_id === id)?.seconds ?? {} }))
    .sort((a, b) => (b.s?.answered ?? 0) - (a.s?.answered ?? 0));
  const extraKeys = Array.from(new Set(times.flatMap((t) => Object.keys(t.seconds ?? {})))).filter((k) => !TIME_KEYS.includes(k) && k !== 'offline');
  const keys = [...TIME_KEYS, ...extraKeys];
  return (
    <div className="table-wrap">
      <table className="table">
        <thead>
          <tr>
            <th>Agent</th><th className="num">Calls</th><th className="num">Answered</th><th className="num">Outbound</th><th className="num">Talk</th><th className="num">Avg talk</th><th className="num">CSAT</th>
            {keys.map((k) => <th key={k} className="num">{k === 'busy' ? 'On call' : k} time</th>)}
            <th className="num">Occupancy</th>
          </tr>
        </thead>
        <tbody>
          {rows.map(({ id, s, t }) => {
            const logged = keys.reduce((n, k) => n + (t[k] ?? 0), 0);
            const occ = logged ? ((t.busy ?? 0) + (t.wrapup ?? 0) + (t.ringing ?? 0)) / logged : 0;
            return (
              <tr key={id}>
                <td className="strong" style={{ whiteSpace: 'nowrap' }}>{name(id)}</td>
                <td className="num">{fmtNum(s?.total ?? 0)}</td><td className="num">{fmtNum(s?.answered ?? 0)}</td><td className="num">{fmtNum(s?.outbound ?? 0)}</td>
                <td className="num">{fmtLongDuration(s?.total_talk_sec ?? 0)}</td><td className="num">{fmtDuration(s?.avg_talk_sec ?? 0)}</td>
                <td className="num" title={s?.csat_count ? `${s.csat_count} ratings` : undefined}>{fmtCsat(s?.csat_avg, s?.csat_count)}</td>
                {keys.map((k) => <td key={k} className="num">{t[k] ? fmtLongDuration(t[k]) : '—'}</td>)}
                <td className="num">{logged ? fmtPct(occ * 100) : '—'}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
