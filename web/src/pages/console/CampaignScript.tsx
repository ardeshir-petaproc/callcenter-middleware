import { Megaphone } from 'lucide-react';
import { useMe } from '../../state/auth';
import { useLookup } from '../../state/lookups';
import { Card } from '../../components/ui/primitives';

/** Campaign name + agent script for outbound campaign calls ({agent} → my display name). */
export function CampaignScript({ campaignId }: { campaignId: number }) {
  const me = useMe();
  const campaigns = useLookup('campaigns');
  const c = campaigns.find((x) => x.id === campaignId);
  const script = (c?.script ?? '').replace(/\{agent\}/g, me.display_name || me.username);
  return (
    <Card title={c ? `Campaign · ${c.name}` : `Campaign #${campaignId}`} icon={<Megaphone />}>
      {c?.description && <p className="small muted" style={{ marginBottom: 10 }}>{c.description}</p>}
      {script
        ? <div className="pre" style={{ background: 'var(--primary-soft)', color: 'var(--primary-text)', padding: 12, borderRadius: 8, fontSize: 15, lineHeight: 1.55 }}>{script}</div>
        : <div className="small muted">No script for this campaign.</div>}
    </Card>
  );
}
