import { useEffect, useRef } from 'react';
import { MessageSquare } from 'lucide-react';
import type { TranscriptEntry } from '../../api/types';
import { fmtTime } from '../../lib/format';
import { Empty } from '../ui/primitives';

const side: Record<string, string> = { caller: 'left', ai: 'right', agent: 'right agent', call: 'center' };
const who: Record<string, string> = { caller: 'Caller', ai: 'AI agent', agent: 'Agent', call: '' };

/** Chat-bubble transcript; auto-scrolls to the newest entry when `follow` is set. */
export function TranscriptView({ entries, follow }: { entries: TranscriptEntry[]; follow?: boolean }) {
  const end = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (follow) end.current?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
  }, [entries.length, follow]);
  if (!entries.length) {
    return <Empty title="No transcript yet" icon={<MessageSquare />}>Transcribed speech appears here as the conversation happens.</Empty>;
  }
  return (
    <div className="transcript">
      {entries.map((e, i) => (
        <div key={`${e.at}-${i}`} className={`bubble-row ${side[e.speaker] ?? 'left'}`}>
          <div className="bubble">{e.text}</div>
          <div className="bubble-meta">{who[e.speaker] ?? e.speaker}{who[e.speaker] ? ' · ' : ''}{fmtTime(e.at)}</div>
        </div>
      ))}
      <div ref={end} />
    </div>
  );
}
