import { useState } from 'react';
import { CheckCircle2, Coffee, PowerOff } from 'lucide-react';
import type { AgentState, AgentStatus } from '../../api/types';
import { post } from '../../api/client';
import { applyAgent } from '../../state/live';
import { useLookup } from '../../state/lookups';
import { useNow } from '../../lib/store';
import { fmtDuration, secsSince } from '../../lib/format';
import { agentStatusLabel } from '../../components/ui/Badges';
import { toast } from '../../components/ui/Toast';

type Settable = 'available' | 'paused' | 'offline';
const OPTIONS: { value: Settable; label: string; icon: typeof Coffee }[] = [
  { value: 'available', label: 'Available', icon: CheckCircle2 },
  { value: 'paused', label: 'Paused', icon: Coffee },
  { value: 'offline', label: 'Offline', icon: PowerOff },
];

export function StatusControl({ agent }: { agent?: AgentState }) {
  const reasons = useLookup('pauseReasons').filter((r) => r.enabled);
  const [picking, setPicking] = useState(false);
  const [busy, setBusy] = useState(false);
  const now = useNow();
  const status: AgentStatus = agent?.status ?? 'offline';

  const set = async (s: Settable, reason?: string) => {
    setBusy(true);
    try {
      const res = await post<AgentState>('/me/status', reason ? { status: s, reason } : { status: s });
      if (res && typeof res === 'object' && 'status' in res) applyAgent(res);
      setPicking(false);
      toast.success(`You are now ${agentStatusLabel(s).toLowerCase()}${reason ? ` (${reason})` : ''}`);
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };

  const onCall = status === 'busy' || status === 'ringing';
  return (
    <div className="col gap-lg">
      <div className={`status-hero st st-${status}`}>
        <span className="big-dot" aria-hidden />
        <span className="label">{agentStatusLabel(status)}</span>
        <span className="timer">{fmtDuration(secsSince(agent?.since, now))}</span>
        {status === 'paused' && agent?.pause_reason && <span className="badge st st-paused">{agent.pause_reason}</span>}
        {status === 'wrapup' && agent?.wrapup_until && (
          <span className="badge st st-wrapup">Wrap-up ends in {fmtDuration(Math.max(0, Math.round((Date.parse(agent.wrapup_until) - now) / 1000)))}</span>
        )}
      </div>
      <div className="status-control" role="group" aria-label="Set my status">
        {OPTIONS.map((o) => {
          const active = status === o.value || (o.value === 'available' && (status === 'busy' || status === 'ringing' || status === 'wrapup'));
          return (
            <button key={o.value} className={`status-btn st st-${o.value} ${status === o.value ? 'active' : ''}`} aria-pressed={active}
              disabled={busy || (onCall && o.value !== 'paused')}
              onClick={() => (o.value === 'paused' ? setPicking((p) => !p) : set(o.value))}>
              <o.icon aria-hidden />{o.label}
            </button>
          );
        })}
      </div>
      {picking && (
        <div className="card" style={{ padding: 12 }}>
          <div className="field-label" style={{ marginBottom: 8 }}>Pause reason</div>
          <div className="btn-group">
            {reasons.map((r) => (
              <button key={r.id} className="btn" disabled={busy} onClick={() => set('paused', r.label)}>{r.label}</button>
            ))}
            <button className="btn ghost" disabled={busy} onClick={() => set('paused')}>No reason</button>
          </div>
          {onCall && <div className="small muted" style={{ marginTop: 8 }}>The pause takes effect after your current call.</div>}
        </div>
      )}
    </div>
  );
}
