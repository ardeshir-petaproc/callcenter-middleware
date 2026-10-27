import { Bot, PhoneCall, PhoneIncoming, PhoneOff, X } from 'lucide-react';
import type { ScreenPop } from '../../api/types';
import { useNow, useStore } from '../../lib/store';
import { clearScreenPop, live } from '../../state/live';
import { useQueueNames } from '../../state/lookups';
import { softphone, answer, hangup } from '../../state/softphone';
import { liveWait } from '../../lib/calls';
import { fmtDuration } from '../../lib/format';
import { stopRinging } from '../../lib/ringtone';
import { ContactPanel } from './ContactPanel';
import { TranscriptView } from '../../components/calls/TranscriptView';

const EMPTY: never[] = [];

/** Incoming-call screen pop: caller, CRM record, IVR path and AI hand-off context. */
export function ScreenPopPanel({ pop }: { pop: ScreenPop }) {
  const c = pop.call;
  const now = useNow();
  const queueName = useQueueNames();
  const transcript = useStore(live, (s) => s.transcripts[c.id] ?? EMPTY);
  const soft = useStore(softphone, (s) => s.call);
  const viaAI = c.ai_profile_id > 0 || transcript.some((t) => t.speaker === 'ai');

  return (
    <section className="card screenpop" aria-live="assertive">
      <div className="card-head">
        <h2><PhoneIncoming style={{ color: 'var(--info-solid)' }} />Incoming call{c.queue_id ? ` · ${queueName(c.queue_id)}` : ''}</h2>
        <button className="btn ghost icon sm" aria-label="Dismiss" onClick={() => { stopRinging(); clearScreenPop(); }}><X /></button>
      </div>
      <div className="card-body col gap-lg">
        <div className="row between wrap">
          <div>
            <div className="caller">{c.caller_name || c.caller_number || 'Unknown caller'}</div>
            {c.caller_name && <div className="muted">{c.caller_number}</div>}
          </div>
          <div className="right">
            <div className="num strong" style={{ fontSize: 18 }}>{fmtDuration(liveWait(c, now))}</div>
            <div className="small muted">waiting</div>
          </div>
        </div>
        <div className="row wrap small">
          {c.ivr_path && <span>IVR path: <code>{c.ivr_path}</code></span>}
          {c.dnis && <span className="muted">Dialed {c.dnis}</span>}
        </div>
        {soft?.direction === 'incoming' && soft.state === 'ringing' ? (
          <div className="btn-group">
            <button className="btn success lg" onClick={answer}><PhoneCall />Answer</button>
            <button className="btn danger lg" onClick={hangup}><PhoneOff />Decline</button>
          </div>
        ) : (
          <div className="alert info small">Your phone is ringing — pick up to connect.</div>
        )}
        {viaAI && (
          <div className="col">
            <div className="field-label row"><Bot size={14} />AI hand-off</div>
            {c.summary ? <div className="pre small" style={{ background: 'var(--violet-soft)', padding: 10, borderRadius: 8 }}>{c.summary}</div> : null}
            {transcript.length > 0 && (
              <div style={{ maxHeight: 220, overflowY: 'auto', border: '1px solid var(--border)', borderRadius: 8, padding: 8 }}>
                <TranscriptView entries={transcript} follow />
              </div>
            )}
          </div>
        )}
        <ContactPanel phone={c.caller_number} excludeCallId={c.id} />
      </div>
    </section>
  );
}
