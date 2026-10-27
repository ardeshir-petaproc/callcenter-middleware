import { useState } from 'react';
import { Bell, BellOff, Headset } from 'lucide-react';
import { Card, PageHeader } from '../../components/ui/primitives';
import { useStore } from '../../lib/store';
import { live } from '../../state/live';
import { useMyAgent, useMyCurrentCall } from '../../state/selectors';
import { useMe } from '../../state/auth';
import { notificationsSupported, playBeep, requestNotifications, unlockAudio } from '../../lib/ringtone';
import { StatusControl } from './StatusControl';
import { ScreenPopPanel } from './ScreenPopPanel';
import { CurrentCallCard } from './CurrentCallCard';
import { WrapUpForm } from './WrapUpForm';
import { DialerCard } from './DialerCard';
import { MyCallbacks } from './MyCallbacks';
import { MyStats } from './MyStats';
import { SoftphoneCard } from './SoftphoneCard';
import { CampaignScript } from './CampaignScript';

function NotifyButton() {
  const [perm, setPerm] = useState(() => (notificationsSupported() ? Notification.permission : 'denied'));
  if (!notificationsSupported()) return null;
  if (perm === 'granted') return <span className="badge ok"><Bell size={12} />Alerts on</span>;
  return (
    <button className="btn sm" disabled={perm === 'denied'} title={perm === 'denied' ? 'Blocked in browser settings' : undefined}
      onClick={async () => { unlockAudio(); playBeep(); const p = await requestNotifications(); if (p !== 'unsupported') setPerm(p); }}>
      {perm === 'denied' ? <BellOff /> : <Bell />}{perm === 'denied' ? 'Notifications blocked' : 'Enable call alerts'}
    </button>
  );
}

export default function ConsolePage() {
  const me = useMe();
  const agent = useMyAgent();
  const pop = useStore(live, (s) => s.screenpop);
  const wrapup = useStore(live, (s) => s.wrapup);
  const current = useMyCurrentCall();

  return (
    <>
      <PageHeader
        title={`Hi, ${(me.display_name || me.username).split(' ')[0]}`}
        sub={agent?.extension || me.extension ? `Extension ${agent?.extension || me.extension}` : 'Agent console'}
        actions={<NotifyButton />}
      />
      <div className="console-grid">
        <div className="stack">
          <Card title="My status" icon={<Headset />}>
            <StatusControl agent={agent} />
          </Card>
          {pop && <ScreenPopPanel pop={pop} />}
          {current && <CurrentCallCard call={current} />}
          {current && current.campaign_id > 0 && <CampaignScript campaignId={current.campaign_id} />}
          {wrapup && !current && <WrapUpForm key={wrapup.id} call={wrapup} />}
          <MyCallbacks />
        </div>
        <div className="stack">
          <MyStats />
          <DialerCard />
          <SoftphoneCard />
        </div>
      </div>
    </>
  );
}
