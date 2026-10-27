import { Fragment, useEffect, type ReactNode } from 'react';
import { Link, useParams } from 'react-router-dom';
import { ArrowLeft, Bot, Clock3, MessageSquare, Mic } from 'lucide-react';
import type { CallDetail } from '../../api/types';
import { authUrl, get } from '../../api/client';
import { useApi } from '../../lib/hooks';
import { useNow, useStore } from '../../lib/store';
import { live, seedTranscript } from '../../state/live';
import { useLookup, useQueueNames, useUserNames } from '../../state/lookups';
import { fmtDateTime, fmtDuration, fmtTime, humanize } from '../../lib/format';
import { callerLabel, liveDuration } from '../../lib/calls';
import { Card, Empty, ErrorBox, PageHeader, SkeletonRows } from '../../components/ui/primitives';
import { CallStateBadge, CsatBadge, OutcomeBadge, SentimentBadge } from '../../components/ui/Badges';
import { TranscriptView } from '../../components/calls/TranscriptView';
import { CallActions } from '../../components/calls/CallActions';
import { CallEditForm } from './CallEditForm';

const EMPTY: never[] = [];

export default function CallDetailPage() {
  const { id = '' } = useParams();
  const { data, loading, error, reload, setData } = useApi((s) => get<CallDetail>(`/calls/${encodeURIComponent(id)}`, undefined, s), [id]);
  const liveCall = useStore(live, (s) => s.calls[id]);
  const liveTranscript = useStore(live, (s) => s.transcripts[id] ?? EMPTY);
  const now = useNow();
  const queueName = useQueueNames();
  const userName = useUserNames();
  const flows = useLookup('flows');
  const profiles = useLookup('aiProfiles');
  const campaigns = useLookup('campaigns');

  useEffect(() => { if (data?.transcript && liveCall) seedTranscript(id, data.transcript); }, [data, liveCall, id]);
  // Reload the final CDR once a live call ends (summary / sentiment arrive later).
  useEffect(() => { if (data?.live && !liveCall) { const t = window.setTimeout(reload, 1500); return () => clearTimeout(t); } }, [data?.live, liveCall, reload]);

  if (error) return <ErrorBox error={error} onRetry={reload} />;
  if (loading && !data) return <div className="card"><SkeletonRows rows={8} /></div>;
  if (!data) return null;
  const c = liveCall ? { ...data.call, ...liveCall } : data.call;
  const transcript = liveCall ? liveTranscript : data.transcript ?? [];
  const flowName = flows.find((f) => f.id === c.flow_id)?.name;
  const aiName = profiles.find((p) => p.id === c.ai_profile_id)?.name;
  const meta: [string, ReactNode][] = [
    ['Direction', humanize(c.direction)], ['Number', c.caller_number || '—'], ['Dialed (DNIS)', c.dnis || '—'],
    ['Started', fmtDateTime(c.started_at)], ['Answered', fmtTime(c.answered_at)], ['Ended', fmtTime(c.ended_at)],
    ['Duration', fmtDuration(liveDuration(c, now))], ['Wait', fmtDuration(c.wait_sec)], ['Talk', fmtDuration(c.talk_sec)],
    ['Hold', fmtDuration(c.hold_sec)], ['Queue', queueName(c.queue_id)], ['Agent', userName(c.agent_id)],
    ['IVR flow', flowName ?? (c.flow_id ? `#${c.flow_id}` : '—')], ['IVR path', c.ivr_path ? <code>{c.ivr_path}</code> : '—'],
    ['AI agent', aiName ?? (c.ai_profile_id ? `#${c.ai_profile_id}` : '—')], ['SLA', c.answered_at ? (c.sla_met ? 'Met' : 'Missed') : '—'],
    ['CSAT', c.csat ? <CsatBadge csat={c.csat} /> : 'Not rated'],
    ['Campaign', c.campaign_id ? <Link to={`/campaigns/${c.campaign_id}`}>{campaigns.find((x) => x.id === c.campaign_id)?.name ?? `#${c.campaign_id}`}</Link> : '—'],
    ['Hung up by', c.hangup_by ? humanize(c.hangup_by) : '—'], ['Call ID', <code style={{ wordBreak: 'break-all' }}>{c.id}</code>],
  ];

  return (
    <>
      <Link to="/calls" className="row small" style={{ gap: 4 }}><ArrowLeft size={14} />Call history</Link>
      <PageHeader
        title={<span className="row wrap">{callerLabel(c)} {liveCall ? <CallStateBadge state={liveCall.state} /> : <OutcomeBadge outcome={c.outcome} />}<SentimentBadge sentiment={c.sentiment} /><CsatBadge csat={c.csat} /></span>}
        sub={`${humanize(c.direction)} call · ${fmtDateTime(c.started_at)}${data.contact ? ` · ${data.contact.name}` : ''}`}
        actions={liveCall && <CallActions call={liveCall} compact />}
      />
      <div className="grid-2" style={{ alignItems: 'start' }}>
        <div className="stack">
          <Card title="Details">
            <dl className="kv">{meta.map(([k, v]) => <Fragment key={k}><dt>{k}</dt><dd>{v}</dd></Fragment>)}</dl>
          </Card>
          {(c.summary || c.sentiment) && (
            <Card title="AI summary" icon={<Bot />} actions={<SentimentBadge sentiment={c.sentiment} />}>
              <p className="pre">{c.summary || 'No summary available.'}</p>
            </Card>
          )}
          <Card title="Recording" icon={<Mic />}>
            {c.recording ? <audio controls preload="none" src={authUrl(`/calls/${encodeURIComponent(c.id)}/recording`)} />
              : <div className="small muted">This call was not recorded.</div>}
          </Card>
          <CallEditForm call={data.call} onSaved={(call) => setData((d) => (d ? { ...d, call } : d))} />
        </div>
        <div className="stack">
          <Card title="Transcript" icon={<MessageSquare />}>
            <div style={{ maxHeight: 520, overflowY: 'auto' }}><TranscriptView entries={transcript} follow={!!liveCall} /></div>
          </Card>
          <Card title="Timeline" icon={<Clock3 />}>
            {data.events?.length ? (
              <ul className="timeline">
                {data.events.map((e, i) => (
                  <li key={i}>
                    <span className="t-type">{humanize(e.type)}</span><span className="t-time">{fmtTime(e.at)}</span>
                    {e.detail && <div className="t-detail">{e.detail}</div>}
                  </li>
                ))}
              </ul>
            ) : <Empty title="No events recorded" />}
          </Card>
        </div>
      </div>
    </>
  );
}
