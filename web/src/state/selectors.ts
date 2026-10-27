import { useMemo } from 'react';
import type { AgentState, CallView } from '../api/types';
import { useStore } from '../lib/store';
import { live } from './live';

export function useMyAgent(): AgentState | undefined {
  const meId = useStore(live, (s) => s.meId);
  const agents = useStore(live, (s) => s.agents);
  return agents[meId];
}

const ACTIVE = new Set(['connected', 'hold', 'dialing', 'ringing']);

/** The call the logged-in agent is currently handling (connected/hold/dialing). */
export function useMyCurrentCall(): CallView | undefined {
  const me = useMyAgent();
  const meId = useStore(live, (s) => s.meId);
  const calls = useStore(live, (s) => s.calls);
  return useMemo(() => {
    if (me?.call_id && calls[me.call_id] && calls[me.call_id].state !== 'ringing') return calls[me.call_id];
    return Object.values(calls).find((c) => c.agent_id === meId && ACTIVE.has(c.state) && c.state !== 'ringing');
  }, [me?.call_id, calls, meId]);
}

export function useLiveCalls(): CallView[] {
  const calls = useStore(live, (s) => s.calls);
  return useMemo(() => Object.values(calls).sort((a, b) => Date.parse(a.started_at) - Date.parse(b.started_at)), [calls]);
}

export function useLiveAgents(): AgentState[] {
  const agents = useStore(live, (s) => s.agents);
  return useMemo(() => {
    const order: Record<string, number> = { ringing: 0, busy: 1, wrapup: 2, available: 3, paused: 4, offline: 5 };
    return Object.values(agents).sort((a, b) => (order[a.status] ?? 9) - (order[b.status] ?? 9)
      || (a.display_name || a.username).localeCompare(b.display_name || b.username));
  }, [agents]);
}

/** Queue ids currently in alert (snapshot `alerts` plus live `queue.alert` events). */
export function useAlertingQueues(): Set<number> {
  const events = useStore(live, (s) => s.queueAlerts);
  return useMemo(() => new Set<number>(Object.keys(events).map(Number)), [events]);
}
