import { useMemo, useState } from 'react';
import { BarChart3, CalendarDays, Clock, Download, ListTree, PieChart, Smile, Users } from 'lucide-react';
import type { AgentTime, Bucket, GroupStat, LabelCount, Summary } from '../../api/types';
import { authUrl, get } from '../../api/client';
import { useApi } from '../../lib/hooks';
import { dayEndIso, dayStartIso, fmtCsat, fmtDuration, fmtNum, fmtPct } from '../../lib/format';
import { useLookup, useQueueNames, useUserNames } from '../../state/lookups';
import { Card, ErrorBox, Kpi, PageHeader, SkeletonRows } from '../../components/ui/primitives';
import { abandonTone, csatTone, slaTone } from '../dashboard/KpiTiles';
import { DateRangePicker, presetRange, type Range } from './DateRange';
import { BreakdownChart, VolumeChart } from './Charts';
import { AgentTable, QueueTable } from './ReportTables';

interface Data {
  summary: Summary; queues: GroupStat[]; agents: GroupStat[]; agentTime: AgentTime[];
  hourly: Bucket[]; daily: Bucket[]; outcome: LabelCount[]; disposition: LabelCount[]; sentiment: LabelCount[];
}

export default function ReportsPage() {
  const [range, setRange] = useState<Range>(() => presetRange('today'));
  const [queueId, setQueueId] = useState('');
  const queues = useLookup('queues');
  const queueName = useQueueNames();
  const userName = useUserNames();
  const q = useMemo(() => ({ from: dayStartIso(range.from), to: dayEndIso(range.to), queue_id: queueId }), [range, queueId]);
  const tz = -new Date().getTimezoneOffset();

  const { data, loading, error, reload } = useApi<Data>(async (signal) => {
    const g = <T,>(p: string, extra?: Record<string, string | number>) => get<T>(p, { ...q, ...extra }, signal).catch(() => [] as unknown as T);
    const [summary, qs, ags, at, hourly, daily, outcome, disposition, sentiment] = await Promise.all([
      get<Summary>('/reports/summary', q, signal),
      g<GroupStat[]>('/reports/queues'), g<GroupStat[]>('/reports/agents'), g<AgentTime[]>('/reports/agent-time'),
      g<Bucket[]>('/reports/timeseries', { by: 'hour', tz_offset: tz }), g<Bucket[]>('/reports/timeseries', { by: 'day', tz_offset: tz }),
      g<LabelCount[]>('/reports/breakdown', { by: 'outcome' }), g<LabelCount[]>('/reports/breakdown', { by: 'disposition' }), g<LabelCount[]>('/reports/breakdown', { by: 'sentiment' }),
    ]);
    const arr = <T,>(x: T[] | null | undefined) => (Array.isArray(x) ? x : []);
    return { summary, queues: arr(qs), agents: arr(ags), agentTime: arr(at), hourly: arr(hourly), daily: arr(daily), outcome: arr(outcome), disposition: arr(disposition), sentiment: arr(sentiment) };
  }, [q.from, q.to, q.queue_id]);

  const s = data?.summary;
  const multiDay = range.from !== range.to;
  return (
    <>
      <PageHeader title="Reports" sub="Performance, service level and agent productivity" actions={
        <a className="btn" href={authUrl('/reports/export.csv', q)} download><Download />Export CSV</a>
      } />
      <div className="row wrap between">
        <DateRangePicker value={range} onChange={setRange} />
        <select className="select sm" style={{ width: 170 }} value={queueId} onChange={(e) => setQueueId(e.target.value)} aria-label="Queue filter">
          <option value="">All queues</option>
          {queues.map((x) => <option key={x.id} value={x.id}>{x.name}</option>)}
        </select>
      </div>
      {error && <ErrorBox error={error} onRetry={reload} />}
      {!data ? (loading && <div className="card"><SkeletonRows rows={6} /></div>) : (
        <div className="stack" style={{ opacity: loading ? 0.6 : 1, transition: 'opacity .15s' }}>
          <div className="kpis">
            <Kpi label="Total calls" value={fmtNum(s?.total)} sub={`${fmtNum(s?.inbound)} in · ${fmtNum(s?.outbound)} out`} />
            <Kpi label="Answered" value={fmtNum(s?.answered)} sub={`${fmtNum(s?.ai_resolved)} resolved by AI`} />
            <Kpi label="Abandoned" value={fmtNum(s?.abandoned)} tone={abandonTone(s)} sub={s?.total ? `${fmtPct(s.abandon_rate)} abandon rate` : undefined} />
            <Kpi label="SLA" value={s?.answered ? fmtPct(s.sla_percent) : '—'} tone={slaTone(s)} sub={`${fmtNum(s?.sla_met)} within target`} />
            <Kpi label="Avg wait" value={fmtDuration(s?.avg_wait_sec)} sub={`Max ${fmtDuration(s?.max_wait_sec)}`} />
            <Kpi label="Avg talk" value={fmtDuration(s?.avg_talk_sec)} sub={`${fmtDuration(s?.total_talk_sec)} total`} />
            <Kpi label="CSAT" value={fmtCsat(s?.csat_avg, s?.csat_count)} tone={csatTone(s)} sub={s?.csat_count ? `${fmtNum(s.csat_count)} ratings · out of 5` : 'No ratings'} />
            <Kpi label="Voicemail / callbacks" value={`${fmtNum(s?.voicemail)} / ${fmtNum(s?.callbacks)}`} />
          </div>
          <div className={multiDay ? 'grid-2' : 'grid'}>
            <Card title="Calls by hour" icon={<Clock />}><VolumeChart data={data.hourly} by="hour" /></Card>
            {multiDay && <Card title="Calls by day" icon={<CalendarDays />}><VolumeChart data={data.daily} by="day" from={range.from} to={range.to} /></Card>}
          </div>
          <div className="grid-3">
            <Card title="Outcomes" icon={<PieChart />}><BreakdownChart data={data.outcome} /></Card>
            <Card title="Dispositions" icon={<BarChart3 />}><BreakdownChart data={data.disposition} /></Card>
            <Card title="Sentiment" icon={<Smile />}><BreakdownChart data={data.sentiment} /></Card>
          </div>
          <Card title="Queues" icon={<ListTree />} flush><QueueTable rows={data.queues} name={queueName} /></Card>
          <Card title="Agents" icon={<Users />} flush><AgentTable stats={data.agents} times={data.agentTime} name={userName} /></Card>
        </div>
      )}
    </>
  );
}
