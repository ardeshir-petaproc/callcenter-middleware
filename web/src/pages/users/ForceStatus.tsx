import { useState } from 'react';
import { Activity } from 'lucide-react';
import type { AgentState } from '../../api/types';
import { post } from '../../api/client';
import { applyAgent } from '../../state/live';
import { useLookup } from '../../state/lookups';
import { useNow } from '../../lib/store';
import { fmtDuration, fmtLongDuration, secsSince } from '../../lib/format';
import { Card } from '../../components/ui/primitives';
import { AgentStatusBadge } from '../../components/ui/Badges';
import { toast } from '../../components/ui/Toast';

/** Supervisor control: view live agent state and force a status change. */
export function ForceStatus({ userId, agent }: { userId: number; agent?: AgentState }) {
  const reasons = useLookup('pauseReasons').filter((r) => r.enabled);
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);
  const now = useNow();
  const force = async (status: 'available' | 'paused' | 'offline') => {
    setBusy(true);
    try {
      const res = await post<AgentState>(`/users/${userId}/status`, status === 'paused' && reason ? { status, reason } : { status });
      if (res && typeof res === 'object' && 'status' in res) applyAgent(res);
      toast.success(`Status set to ${status}`);
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  return (
    <Card title="Live status" icon={<Activity />}>
      <div className="col gap-lg">
        <div className="row wrap">
          <AgentStatusBadge status={agent?.status ?? 'offline'} reason={agent?.pause_reason} />
          {agent && <span className="small muted">for {fmtDuration(secsSince(agent.since, now))} · {agent.calls_today} calls · {fmtLongDuration(agent.talk_secs_today)} talk today</span>}
        </div>
        <div className="row wrap">
          <button className="btn sm" disabled={busy} onClick={() => force('available')}>Set available</button>
          <select className="select sm" style={{ width: 160 }} value={reason} onChange={(e) => setReason(e.target.value)} aria-label="Pause reason">
            <option value="">Pause reason…</option>
            {reasons.map((r) => <option key={r.id} value={r.label}>{r.label}</option>)}
          </select>
          <button className="btn sm" disabled={busy} onClick={() => force('paused')}>Pause</button>
          <button className="btn sm" disabled={busy} onClick={() => force('offline')}>Log out</button>
        </div>
      </div>
    </Card>
  );
}
