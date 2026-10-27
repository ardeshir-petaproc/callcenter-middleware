import { Bot, Star, Clock, Coffee, Headset, PhoneCall, Hourglass, Target, PhoneMissed, Timer, MessageCircle } from 'lucide-react';
import type { Summary } from '../../api/types';
import { Kpi } from '../../components/ui/primitives';
import { fmtCsat, fmtDuration, fmtNum, fmtPct, pctValue } from '../../lib/format';
import type { LiveKpis } from './useKpis';

export function slaTone(s?: Summary): 'ok' | 'warn' | 'danger' | undefined {
  if (!s || !s.answered) return undefined;
  const v = pctValue(s.sla_percent);
  return v >= 80 ? 'ok' : v >= 60 ? 'warn' : 'danger';
}
export function csatTone(s?: Summary): 'ok' | 'warn' | 'danger' | undefined {
  if (!s?.csat_count) return undefined;
  return s.csat_avg >= 4 ? 'ok' : s.csat_avg >= 3 ? 'warn' : 'danger';
}
export function abandonTone(s?: Summary): 'ok' | 'warn' | 'danger' | undefined {
  if (!s || !s.total) return undefined;
  const v = pctValue(s.abandon_rate);
  return v <= 5 ? 'ok' : v <= 10 ? 'warn' : 'danger';
}

export function KpiTiles({ k, s }: { k: LiveKpis; s?: Summary }) {
  return (
    <div className="kpis">
      <Kpi icon={<Hourglass />} label="Calls waiting" value={fmtNum(k.waiting)} tone={k.waiting > 0 ? (k.available ? 'warn' : 'danger') : undefined} />
      <Kpi icon={<Clock />} label="Longest wait" value={k.longestWait ? fmtDuration(k.longestWait) : '—'} tone={k.longestWait > 120 ? 'danger' : k.longestWait > 45 ? 'warn' : undefined} />
      <Kpi icon={<PhoneCall />} label="Active calls" value={fmtNum(k.active)} />
      <Kpi icon={<Headset />} label="Agents available" value={fmtNum(k.available)} tone={k.available ? 'ok' : undefined} sub={`${k.loggedIn} logged in`} />
      <Kpi icon={<MessageCircle />} label="On call / wrap-up" value={`${k.busy} / ${k.wrapup}`} />
      <Kpi icon={<Coffee />} label="Paused" value={fmtNum(k.paused)} />
      <Kpi icon={<Bot />} label="AI sessions" value={fmtNum(k.aiSessions)} />
      <Kpi icon={<Target />} label="SLA today" value={s && s.answered ? fmtPct(s.sla_percent) : '—'} tone={slaTone(s)} sub={s ? `${fmtNum(s.sla_met)} of ${fmtNum(s.answered)} answered` : undefined} />
      <Kpi icon={<PhoneMissed />} label="Abandon rate" value={s && s.total ? fmtPct(s.abandon_rate) : '—'} tone={abandonTone(s)} sub={s ? `${fmtNum(s.abandoned)} of ${fmtNum(s.total)} calls` : undefined} />
      <Kpi icon={<Star />} label="CSAT today" value={s ? fmtCsat(s.csat_avg, s.csat_count) : '—'} tone={csatTone(s)} sub={s?.csat_count ? `${fmtNum(s.csat_count)} rating${s.csat_count === 1 ? '' : 's'} · out of 5` : 'No ratings yet'} />
      <Kpi icon={<Timer />} label="Avg wait / talk" value={s ? `${fmtDuration(s.avg_wait_sec)}` : '—'} sub={s ? `Talk ${fmtDuration(s.avg_talk_sec)}` : undefined} />
    </div>
  );
}
