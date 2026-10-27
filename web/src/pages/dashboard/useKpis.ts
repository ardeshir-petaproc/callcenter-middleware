import { useEffect, useMemo, useState } from 'react';
import type { Summary } from '../../api/types';
import { get } from '../../api/client';
import { useNow, useStore } from '../../lib/store';
import { live } from '../../state/live';
import { onLive } from '../../state/ws';
import { useLiveCalls } from '../../state/selectors';
import { secsSince } from '../../lib/format';

export interface LiveKpis {
  waiting: number; longestWait: number; active: number; available: number; busy: number; paused: number;
  wrapup: number; loggedIn: number; aiSessions: number;
}

export function useLiveKpis(): LiveKpis {
  const calls = useLiveCalls();
  const agents = useStore(live, (s) => s.agents);
  const queues = useStore(live, (s) => s.queues);
  const aiSessions = useStore(live, (s) => s.aiSessions);
  const now = useNow();
  return useMemo(() => {
    const queued = calls.filter((c) => c.state === 'queued');
    const liveLongest = queued.reduce((m, c) => Math.max(m, secsSince(c.queued_at ?? c.started_at, now)), 0);
    const qLongest = queues.reduce((m, q) => Math.max(m, q.longest_wait_sec), 0);
    const counts = { available: 0, busy: 0, paused: 0, wrapup: 0, loggedIn: 0 };
    for (const a of Object.values(agents)) {
      if (a.status === 'available') counts.available++;
      else if (a.status === 'busy' || a.status === 'ringing') counts.busy++;
      else if (a.status === 'paused') counts.paused++;
      else if (a.status === 'wrapup') counts.wrapup++;
      if (a.status !== 'offline') counts.loggedIn++;
    }
    return {
      waiting: queued.length || queues.reduce((n, q) => n + q.waiting, 0),
      longestWait: queued.length ? liveLongest : qLongest,
      active: calls.length,
      aiSessions,
      ...counts,
    };
  }, [calls, agents, queues, aiSessions, now]);
}

/** Today's summary from /reports/summary, refreshed periodically and after calls end. */
export function useTodaySummary(intervalMs = 30000): Summary | undefined {
  const [data, setData] = useState<Summary>();
  useEffect(() => {
    let timer: number | undefined;
    let pending: number | undefined;
    const load = () => get<Summary>('/reports/summary').then(setData).catch(() => undefined);
    void load();
    timer = window.setInterval(load, intervalMs);
    const off = onLive('call.ended', () => {
      if (pending) return;
      pending = window.setTimeout(() => { pending = undefined; void load(); }, 3000);
    });
    return () => { clearInterval(timer); if (pending) clearTimeout(pending); off(); };
  }, [intervalMs]);
  return data;
}
