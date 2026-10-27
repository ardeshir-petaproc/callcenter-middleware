import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Activity, Maximize2, Users } from 'lucide-react';
import { Card, PageHeader, SkeletonRows } from '../../components/ui/primitives';
import { LiveCallsTable } from '../../components/calls/LiveCallsTable';
import { TranscriptDrawer } from '../../components/calls/TranscriptDrawer';
import { useStore } from '../../lib/store';
import { live } from '../../state/live';
import { useLiveAgents, useLiveCalls } from '../../state/selectors';
import { useLocalStorage } from '../../lib/hooks';
import { KpiTiles } from './KpiTiles';
import { QueueCards } from './QueueCards';
import { AgentBoard } from './AgentBoard';
import { useLiveKpis, useTodaySummary } from './useKpis';

export default function DashboardPage() {
  const k = useLiveKpis();
  const summary = useTodaySummary();
  const queues = useStore(live, (s) => s.queues);
  const ready = useStore(live, (s) => s.hasSnapshot);
  const calls = useLiveCalls();
  const agents = useLiveAgents();
  const [openCall, setOpenCall] = useState<string | null>(null);
  const [hideOffline, setHideOffline] = useLocalStorage('cc.dash.hideOffline', true);

  return (
    <>
      <PageHeader
        title="Dashboard"
        sub="Real-time operations overview"
        actions={<Link className="btn" to="/wallboard"><Maximize2 />Wallboard</Link>}
      />
      <KpiTiles k={k} s={summary} />
      {ready ? <QueueCards queues={queues} /> : <div className="card"><SkeletonRows rows={3} /></div>}
      <Card title={`Live calls (${calls.length})`} icon={<Activity />} flush actions={<Link className="btn sm" to="/live">View all</Link>}>
        {ready ? <LiveCallsTable calls={calls} limit={8} onOpen={setOpenCall} /> : <SkeletonRows rows={4} />}
      </Card>
      <Card
        title="Agent board"
        icon={<Users />}
        actions={
          <label className="checkbox small">
            <input type="checkbox" checked={hideOffline} onChange={(e) => setHideOffline(e.target.checked)} /> Hide offline
          </label>
        }
      >
        {ready ? <AgentBoard agents={agents} hideOffline={hideOffline} /> : <SkeletonRows rows={3} />}
      </Card>
      <TranscriptDrawer callId={openCall} onClose={() => setOpenCall(null)} />
    </>
  );
}
