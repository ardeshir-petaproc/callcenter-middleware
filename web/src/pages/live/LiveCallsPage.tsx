import { useMemo, useState } from 'react';
import { Search } from 'lucide-react';
import { Card, PageHeader, SkeletonRows } from '../../components/ui/primitives';
import { LiveCallsTable } from '../../components/calls/LiveCallsTable';
import { TranscriptDrawer } from '../../components/calls/TranscriptDrawer';
import { useLiveCalls } from '../../state/selectors';
import { useLookup } from '../../state/lookups';
import { useStore } from '../../lib/store';
import { live } from '../../state/live';
import type { CallState } from '../../api/types';

const GROUPS: { value: string; label: string; states: CallState[] }[] = [
  { value: 'all', label: 'All', states: [] },
  { value: 'waiting', label: 'Waiting', states: ['queued', 'ringing'] },
  { value: 'talking', label: 'Talking', states: ['connected', 'hold', 'dialing'] },
  { value: 'self', label: 'IVR / AI', states: ['ivr', 'ai', 'voicemail', 'message'] },
];

export default function LiveCallsPage() {
  const calls = useLiveCalls();
  const ready = useStore(live, (s) => s.hasSnapshot);
  const queues = useLookup('queues');
  const [group, setGroup] = useState('all');
  const [queue, setQueue] = useState('');
  const [q, setQ] = useState('');
  const [openCall, setOpenCall] = useState<string | null>(null);

  const counts = useMemo(() => Object.fromEntries(GROUPS.map((g) => [g.value, g.states.length ? calls.filter((c) => g.states.includes(c.state)).length : calls.length])), [calls]);
  const filtered = useMemo(() => {
    const g = GROUPS.find((x) => x.value === group)!;
    const needle = q.trim().toLowerCase();
    return calls.filter((c) => (!g.states.length || g.states.includes(c.state))
      && (!queue || String(c.queue_id) === queue)
      && (!needle || `${c.caller_name} ${c.caller_number} ${c.dnis} ${c.ivr_path}`.toLowerCase().includes(needle)));
  }, [calls, group, queue, q]);

  return (
    <>
      <PageHeader title="Live calls" sub="Every call in progress, updated in real time" />
      <Card flush title={
        <div className="segmented" role="group" aria-label="Filter by state">
          {GROUPS.map((g) => (
            <button key={g.value} className={group === g.value ? 'active' : ''} aria-pressed={group === g.value} onClick={() => setGroup(g.value)}>
              {g.label} <span className="muted num">{counts[g.value]}</span>
            </button>
          ))}
        </div>
      } actions={
        <>
          <select className="select sm" style={{ width: 150 }} value={queue} onChange={(e) => setQueue(e.target.value)} aria-label="Queue">
            <option value="">All queues</option>
            {queues.map((x) => <option key={x.id} value={x.id}>{x.name}</option>)}
          </select>
          <div className="input-icon" style={{ width: 200 }}>
            <Search />
            <input className="input sm" placeholder="Search caller…" value={q} onChange={(e) => setQ(e.target.value)} aria-label="Search calls" />
          </div>
        </>
      }>
        {ready ? <LiveCallsTable calls={filtered} onOpen={setOpenCall} /> : <SkeletonRows rows={6} />}
      </Card>
      <TranscriptDrawer callId={openCall} onClose={() => setOpenCall(null)} />
    </>
  );
}
