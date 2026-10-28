import { useSearchParams } from 'react-router-dom';
import type { Disposition, PauseReason } from '../../api/types';
import { PageHeader, Tabs } from '../../components/ui/primitives';
import { InlineCrud } from './InlineCrud';
import { WebhooksTab } from './WebhooksTab';
import { GeneralTab } from './GeneralTab';
import { AuditTab, SystemTab } from './SystemTab';
import { SlaTab } from './SlaTab';
import { EmailTab } from './EmailTab';
import { InboundKeyCard } from './InboundKeyCard';
import { RetentionTab } from './RetentionTab';

type Tab = 'general' | 'dispositions' | 'pause' | 'sla' | 'email' | 'retention' | 'webhooks' | 'system' | 'audit';
const TABS: { value: Tab; label: string }[] = [
  { value: 'general', label: 'General' }, { value: 'dispositions', label: 'Dispositions' }, { value: 'pause', label: 'Pause reasons' },
  { value: 'sla', label: 'Ticket SLAs' }, { value: 'email', label: 'Email & messages' }, { value: 'retention', label: 'Retention & export' },
  { value: 'webhooks', label: 'Webhooks' }, { value: 'system', label: 'System' }, { value: 'audit', label: 'Audit log' },
];

export default function SettingsPage() {
  const [params, setParams] = useSearchParams();
  const tab = (TABS.some((t) => t.value === params.get('tab')) ? params.get('tab') : 'general') as Tab;
  return (
    <>
      <PageHeader title="Settings" sub="Organization-wide configuration" />
      <Tabs value={tab} tabs={TABS} onChange={(t) => setParams({ tab: t }, { replace: true })} />
      {tab === 'general' && <GeneralTab />}
      {tab === 'dispositions' && (
        <InlineCrud<Disposition> title="Call dispositions" path="/dispositions" lookup="dispositions"
          cols={[{ key: 'code', label: 'Code', placeholder: 'e.g. sale', width: 200 }, { key: 'label', label: 'Label', placeholder: 'e.g. Sale closed' }]}
          blank={{ code: '', label: '', enabled: true }} describe={(d) => d.label} />
      )}
      {tab === 'pause' && (
        <InlineCrud<PauseReason> title="Pause reasons" path="/pause-reasons" lookup="pauseReasons"
          cols={[{ key: 'label', label: 'Label', placeholder: 'e.g. Lunch' }]} blank={{ label: '', enabled: true }} describe={(p) => p.label} />
      )}
      {tab === 'sla' && <SlaTab />}
      {tab === 'email' && <div className="stack"><EmailTab /><InboundKeyCard /></div>}
      {tab === 'retention' && <RetentionTab />}
      {tab === 'webhooks' && <WebhooksTab />}
      {tab === 'system' && <SystemTab />}
      {tab === 'audit' && <AuditTab />}
    </>
  );
}
