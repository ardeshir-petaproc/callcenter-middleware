import { useSearchParams } from 'react-router-dom';
import { PERMS } from '../../lib/perms';
import { useCan } from '../../state/auth';
import { PageHeader, Tabs } from '../../components/ui/primitives';
import { RulesTab } from './RulesTab';
import { TopicsTab } from './TopicsTab';
import { SimulatorTab } from './SimulatorTab';
import { DecisionsTab } from './DecisionsTab';

type Tab = 'rules' | 'topics' | 'simulator' | 'decisions';

export default function RoutingPage() {
  const can = useCan();
  const manage = can(PERMS.routingManage);
  const tabs: { value: Tab; label: string }[] = [
    { value: 'rules', label: 'Rules' }, { value: 'topics', label: 'Topics' },
    ...(manage ? [{ value: 'simulator' as Tab, label: 'Simulator' }] : []),
    ...(manage || can(PERMS.reportsView) ? [{ value: 'decisions' as Tab, label: 'Decisions' }] : []),
  ];
  const [params, setParams] = useSearchParams();
  const tab = (tabs.some((t) => t.value === params.get('tab')) ? params.get('tab') : 'rules') as Tab;
  return (
    <>
      <PageHeader title="Routing" sub="Explainable rules that send each request to the right person — approved before they take effect" />
      <Tabs value={tab} tabs={tabs} onChange={(t) => setParams({ tab: t }, { replace: true })} />
      {tab === 'rules' && <RulesTab />}
      {tab === 'topics' && <TopicsTab />}
      {tab === 'simulator' && <SimulatorTab />}
      {tab === 'decisions' && <DecisionsTab />}
    </>
  );
}
