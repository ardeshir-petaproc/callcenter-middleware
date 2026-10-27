import type { Call, CallView } from '../api/types';
import { secsSince } from './format';

const WAITING = new Set(['queued', 'ringing']);

export function liveDuration(c: Call, now: number): number {
  if (c.ended_at) return Math.max(0, Math.floor((Date.parse(c.ended_at) - Date.parse(c.started_at)) / 1000));
  return secsSince(c.started_at, now);
}

export function liveWait(c: Call | CallView, now: number): number {
  if (WAITING.has(c.state) && c.queued_at && !c.answered_at) return secsSince(c.queued_at, now);
  if ('waiting_sec' in c && c.waiting_sec && !c.answered_at) return c.waiting_sec;
  return c.wait_sec ?? 0;
}

export function liveTalk(c: Call, now: number): number {
  if (!c.answered_at) return 0;
  if (c.ended_at) return c.talk_sec;
  return secsSince(c.answered_at, now);
}

export function callerLabel(c: Pick<Call, 'caller_name' | 'caller_number' | 'direction' | 'dnis'>): string {
  if (c.direction === 'outbound') return c.dnis || c.caller_number || 'Unknown';
  return c.caller_name || c.caller_number || 'Unknown caller';
}

export function callerSub(c: Pick<Call, 'caller_name' | 'caller_number' | 'direction' | 'dnis'>): string {
  if (c.direction === 'outbound') return 'Outbound';
  return c.caller_name ? c.caller_number : '';
}

export function isActiveForControls(c: Call): boolean {
  return c.state !== 'ended';
}
