import { BellRing, ListTree } from 'lucide-react';
import type { QueueView } from '../../api/types';
import { fmtDuration } from '../../lib/format';
import { Empty } from '../../components/ui/primitives';
import { useStore } from '../../lib/store';
import { live } from '../../state/live';

export function QueueCards({ queues }: { queues: QueueView[] }) {
  const alerts = useStore(live, (s) => s.queueAlerts);
  if (!queues.length) return <div className="card"><Empty title="No queues" icon={<ListTree />}>Create a queue to start routing calls.</Empty></div>;
  return (
    <div className="queue-cards">
      {queues.map((q) => {
        const total = Math.max(1, q.available + q.busy + q.paused);
        const hot = q.waiting > 0 && q.available === 0;
        return (
          <div key={q.id} className={`card queue-card ${hot || q.longest_wait_sec > 120 ? 'alert-wait' : ''} ${alerts[q.id] ? 'queue-alert' : ''}`}>
            <div className="row between">
              <h3 className="ellipsis row" style={{ gap: 6 }}>{alerts[q.id] && <BellRing size={15} style={{ color: 'var(--danger-solid)' }} aria-label="Alert" />}{q.name}</h3>
              <span className="small muted">{q.logged_in}/{q.members} logged in</span>
            </div>
            <div className="row between" style={{ alignItems: 'flex-end' }}>
              <div>
                <div className="big" style={q.waiting > 0 ? { color: hot ? 'var(--danger)' : 'var(--warn)' } : undefined}>{q.waiting}</div>
                <div className="small muted">waiting</div>
              </div>
              <div className="right">
                <div className="strong num" style={{ fontSize: 18 }}>{q.longest_wait_sec ? fmtDuration(q.longest_wait_sec) : '—'}</div>
                <div className="small muted">longest wait</div>
              </div>
            </div>
            <div className="bar" aria-label={`${q.available} available, ${q.busy} busy, ${q.paused} paused`}>
              <span style={{ width: `${(q.available / total) * 100}%`, background: 'var(--ok-solid)' }} />
              <span style={{ width: `${(q.busy / total) * 100}%`, background: 'var(--danger-solid)' }} />
              <span style={{ width: `${(q.paused / total) * 100}%`, background: 'var(--warn-solid)' }} />
            </div>
            <div className="stats">
              <div><b style={{ color: 'var(--ok)' }}>{q.available}</b>Available</div>
              <div><b style={{ color: 'var(--danger)' }}>{q.busy}</b>Busy</div>
              <div><b style={{ color: 'var(--warn)' }}>{q.paused}</b>Paused</div>
            </div>
          </div>
        );
      })}
    </div>
  );
}
