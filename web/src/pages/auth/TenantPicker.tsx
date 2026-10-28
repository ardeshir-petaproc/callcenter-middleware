import { Building2, ChevronRight } from 'lucide-react';
import type { TenantChoice } from '../../api/types_org';
import { RoleBadge } from '../../components/ui/Badges';

/** Organization chooser shown when one Joow account works for several organizations. */
export function TenantPicker({ tenants, onPick, busy, name }: { tenants: TenantChoice[]; onPick: (t: TenantChoice) => void; busy: boolean; name?: string }) {
  return (
    <div className="col" style={{ gap: 10 }}>
      <div>
        <div className="strong">Choose an organization</div>
        <div className="small muted">{name ? <><span dir="auto">{name}</span>, your</> : 'Your'} Joow account works for several organizations.</div>
      </div>
      <div className="list" style={{ border: '1px solid var(--border)', borderRadius: 8 }}>
        {tenants.map((t) => (
          <button key={t.id} type="button" className="list-item clickable plain-button" disabled={busy} onClick={() => onPick(t)}>
            <Building2 size={18} className="muted" />
            <div className="grow" style={{ textAlign: 'left' }}>
              <div className="strong" dir="auto">{t.name}</div>
              <div className="small muted mono">{t.slug}</div>
            </div>
            <RoleBadge role={t.role} />
            <ChevronRight size={16} className="muted" />
          </button>
        ))}
      </div>
    </div>
  );
}
