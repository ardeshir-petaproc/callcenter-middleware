import { useSearchParams } from 'react-router-dom';
import { PageHeader, Tabs } from '../../components/ui/primitives';
import { EvaluationsTab } from './EvaluationsTab';
import { ScorecardsTab } from './ScorecardsTab';
import { SummaryTab } from './SummaryTab';

type Tab = 'evaluations' | 'scorecards' | 'summary';
const TABS: { value: Tab; label: string }[] = [{ value: 'evaluations', label: 'Evaluations' }, { value: 'scorecards', label: 'Scorecards' }, { value: 'summary', label: 'Summary' }];

export default function QualityPage() {
  const [params, setParams] = useSearchParams();
  const tab = (TABS.some((t) => t.value === params.get('tab')) ? params.get('tab') : 'evaluations') as Tab;
  return (
    <>
      <PageHeader title="Quality" sub="Scorecards, call evaluations and coaching" />
      <Tabs value={tab} tabs={TABS} onChange={(t) => setParams({ tab: t }, { replace: true })} />
      {tab === 'evaluations' && <EvaluationsTab />}
      {tab === 'scorecards' && <ScorecardsTab />}
      {tab === 'summary' && <SummaryTab />}
    </>
  );
}
