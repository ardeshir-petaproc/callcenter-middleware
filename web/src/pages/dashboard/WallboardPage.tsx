import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Maximize, Minimize, X } from 'lucide-react';
import { useNow, useStore } from '../../lib/store';
import { live } from '../../state/live';
import { useLiveAgents } from '../../state/selectors';
import { fmtCsat, fmtDuration, fmtPct, fmtNum } from '../../lib/format';
import { AgentBoard } from './AgentBoard';
import { useLiveKpis, useTodaySummary } from './useKpis';
import { abandonTone, csatTone, slaTone } from './KpiTiles';

function Big({ label, value, tone }: { label: string; value: string; tone?: string }) {
  return <div className={`wb-kpi ${tone ?? ''}`}><div className="label">{label}</div><div className="value">{value}</div></div>;
}

/** TV-friendly fullscreen view: huge numbers, live via WebSocket, summary refresh every 15 s. */
export default function WallboardPage() {
  const k = useLiveKpis();
  const s = useTodaySummary(15000);
  const queues = useStore(live, (st) => st.queues);
  const conn = useStore(live, (st) => st.conn);
  const alerts = useStore(live, (st) => st.queueAlerts);
  const agents = useLiveAgents();
  const now = useNow();
  const nav = useNavigate();
  const [fs, setFs] = useState(!!document.fullscreenElement);

  useEffect(() => {
    const fn = () => setFs(!!document.fullscreenElement);
    document.addEventListener('fullscreenchange', fn);
    return () => document.removeEventListener('fullscreenchange', fn);
  }, []);
  const toggleFs = () => {
    if (document.fullscreenElement) void document.exitFullscreen();
    else void document.documentElement.requestFullscreen?.().catch(() => undefined);
  };

  return (
    <div className="wallboard">
      <div className="wb-head">
        <div className="row" style={{ gap: 12 }}>
          <h1 style={{ fontSize: 26 }}>Call Center Live</h1>
          <span className={`conn ${conn}`}><span className="dot" />{conn === 'open' ? 'Live' : 'Reconnecting…'}</span>
        </div>
        <div className="row" style={{ gap: 12 }}>
          <span className="wb-clock">{new Date(now).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' })}</span>
          <button className="btn icon" onClick={toggleFs} aria-label={fs ? 'Exit fullscreen' : 'Enter fullscreen'}>{fs ? <Minimize /> : <Maximize />}</button>
          <button className="btn icon" onClick={() => nav('/dashboard')} aria-label="Close wallboard"><X /></button>
        </div>
      </div>
      <div className="wb-kpis">
        <Big label="Waiting" value={fmtNum(k.waiting)} tone={k.waiting ? (k.available ? 'warn' : 'danger') : undefined} />
        <Big label="Longest wait" value={k.longestWait ? fmtDuration(k.longestWait) : '0:00'} tone={k.longestWait > 120 ? 'danger' : k.longestWait > 45 ? 'warn' : undefined} />
        <Big label="Active calls" value={fmtNum(k.active)} />
        <Big label="Available" value={fmtNum(k.available)} tone={k.available ? 'ok' : 'danger'} />
        <Big label="SLA today" value={s && s.answered ? fmtPct(s.sla_percent) : '—'} tone={slaTone(s)} />
        <Big label="Abandoned" value={s && s.total ? fmtPct(s.abandon_rate) : '—'} tone={abandonTone(s)} />
        <Big label="Answered today" value={s ? fmtNum(s.answered) : '—'} />
        <Big label="CSAT today" value={s ? fmtCsat(s.csat_avg, s.csat_count) : '—'} tone={csatTone(s)} />
        <Big label="Avg wait" value={s ? fmtDuration(s.avg_wait_sec) : '—'} />
      </div>
      <div className="wb-queues">
        {queues.map((q) => (
          <div key={q.id} className={`wb-queue ${alerts[q.id] ? 'queue-alert' : ''}`}>
            <div className="name">{q.name}</div>
            <div className="nums">
              <div><b style={q.waiting ? { color: q.available ? 'var(--warn)' : 'var(--danger)' } : undefined}>{q.waiting}</b>waiting</div>
              <div><b>{q.longest_wait_sec ? fmtDuration(q.longest_wait_sec) : '0:00'}</b>longest</div>
              <div><b style={{ color: 'var(--ok)' }}>{q.available}</b>available</div>
              <div><b>{q.busy}</b>busy</div>
            </div>
          </div>
        ))}
      </div>
      <AgentBoard agents={agents} hideOffline />
    </div>
  );
}
