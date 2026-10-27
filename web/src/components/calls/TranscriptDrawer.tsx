import { useEffect } from 'react';
import { Link } from 'react-router-dom';
import { ExternalLink } from 'lucide-react';
import { Drawer } from '../ui/Dialog';
import { TranscriptView } from './TranscriptView';
import { CallStateBadge } from '../ui/Badges';
import { useStore, useNow } from '../../lib/store';
import { live, seedTranscript } from '../../state/live';
import { get } from '../../api/client';
import type { CallDetail } from '../../api/types';
import { callerLabel, liveDuration } from '../../lib/calls';
import { fmtDuration } from '../../lib/format';

const EMPTY: never[] = [];

/** Side drawer streaming a call's transcript from realtime events (seeded from /calls/{id}). */
export function TranscriptDrawer({ callId, onClose }: { callId: string | null; onClose: () => void }) {
  const entries = useStore(live, (s) => (callId ? s.transcripts[callId] ?? EMPTY : EMPTY));
  const call = useStore(live, (s) => (callId ? s.calls[callId] : undefined));
  const now = useNow();

  useEffect(() => {
    if (!callId) return;
    const ctl = new AbortController();
    get<CallDetail>(`/calls/${encodeURIComponent(callId)}`, undefined, ctl.signal)
      .then((d) => seedTranscript(callId, d.transcript ?? []))
      .catch(() => undefined);
    return () => ctl.abort();
  }, [callId]);

  return (
    <Drawer
      open={!!callId}
      onClose={onClose}
      title={call ? callerLabel(call) : 'Transcript'}
      actions={callId && <Link className="btn ghost sm" to={`/calls/${encodeURIComponent(callId)}`}><ExternalLink />Details</Link>}
    >
      {call && (
        <div className="row wrap small muted">
          <CallStateBadge state={call.state} />
          <span className="num">{fmtDuration(liveDuration(call, now))}</span>
          {call.ivr_path && <span>IVR: <code>{call.ivr_path}</code></span>}
        </div>
      )}
      {!call && callId && <div className="small muted">This call has ended.</div>}
      <TranscriptView entries={entries} follow />
    </Drawer>
  );
}
