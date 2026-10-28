import { useEffect, useState } from 'react';
import { ArrowLeftRight } from 'lucide-react';
import { get } from '../../api/client';
import type { TenantChoice } from '../../api/types_org';
import { useAuth } from '../../state/auth';
import { toast } from '../ui/Toast';

/** Menu entries to switch to another organization of the same Joow account (shown only when there is one). */
export function OrgSwitcher({ onDone }: { onDone: () => void }) {
  const { tenant, switchTenant, user } = useAuth();
  const [list, setList] = useState<TenantChoice[]>([]);
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    if (!user?.joow_account_id) return;
    get<TenantChoice[]>('/auth/tenants').then((l) => setList(l ?? [])).catch(() => setList([]));
  }, [user?.joow_account_id]);
  const others = list.filter((t) => t.id !== tenant?.id);
  if (!others.length) return null;
  const go = async (t: TenantChoice) => {
    setBusy(true);
    try { await switchTenant(t.id); onDone(); } catch (e) { toast.error(e); setBusy(false); }
  };
  return (
    <>
      <div className="menu-head small muted" style={{ paddingTop: 6, paddingBottom: 2 }}>Switch organization</div>
      {others.map((t) => (
        <button key={t.id} role="menuitem" disabled={busy} onClick={() => go(t)}>
          <ArrowLeftRight /><span className="grow" dir="auto">{t.name}</span><span className="small muted">{t.role}</span>
        </button>
      ))}
    </>
  );
}
