import { PhoneCall } from 'lucide-react';
import type { CallView } from '../../api/types';
import { useNow, useStore } from '../../lib/store';
import { live } from '../../state/live';
import { useQueueNames } from '../../state/lookups';
import { callerLabel, liveTalk, liveDuration } from '../../lib/calls';
import { fmtDuration } from '../../lib/format';
import { CallStateBadge } from '../../components/ui/Badges';
import { CallActions } from '../../components/calls/CallActions';
import { TranscriptView } from '../../components/calls/TranscriptView';
import { ContactPanel } from './ContactPanel';

const EMPTY: never[] = [];

export function CurrentCallCard({ call }: { call: CallView }) {
  const now = useNow();
  const queueName = useQueueNames();
  const transcript = useStore(live, (s) => s.transcripts[call.id] ?? EMPTY);
  const talk = call.answered_at ? liveTalk(call, now) : liveDuration(call, now);
  const phone = call.direction === 'outbound' ? call.dnis : call.caller_number;
  return (
    <section className="card" style={{ borderColor: 'var(--ok-solid)' }}>
      <div className="card-head">
        <h2><PhoneCall style={{ color: 'var(--ok-solid)' }} />Current call</h2>
        <CallStateBadge state={call.state} />
      </div>
      <div className="card-body col gap-lg">
        <div className="row between wrap">
          <div>
            <div style={{ fontSize: 20, fontWeight: 700 }}>{callerLabel(call)}</div>
            <div className="small muted">
              {call.direction === 'outbound' ? 'Outbound call' : call.caller_name ? call.caller_number : 'Inbound call'}
              {call.queue_id ? ` · ${queueName(call.queue_id)}` : ''}
              {call.ivr_path ? ` · IVR ${call.ivr_path}` : ''}
            </div>
          </div>
          <div className="call-timer" aria-label="Talk time">{fmtDuration(talk)}</div>
        </div>
        <CallActions call={call} monitor={false} />
        {transcript.length > 0 && (
          <div style={{ maxHeight: 220, overflowY: 'auto', border: '1px solid var(--border)', borderRadius: 8, padding: 8 }}>
            <TranscriptView entries={transcript} follow />
          </div>
        )}
        <ContactPanel phone={phone} excludeCallId={call.id} />
      </div>
    </section>
  );
}
