import { useState } from 'react';
import { Globe, Pause, Play, Plus } from 'lucide-react';
import { get, put } from '../../api/client';
import type { Tenant } from '../../api/types';
import { useApi } from '../../lib/hooks';
import { fmtDateTime } from '../../lib/format';
import { useAuth } from '../../state/auth';
import { Card, Empty, ErrorBox, PageHeader, SkeletonRows } from '../../components/ui/primitives';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';
import { CreateTenantDialog } from './CreateTenantDialog';

/** Platform administration: every organization on this installation. */
export default function PlatformPage() {
  const { tenant: current } = useAuth();
  const { data, loading, error, reload, setData } = useApi((s) => get<Tenant[]>('/platform/tenants', undefined, s), []);
  const [creating, setCreating] = useState(false);
  const setStatus = async (t: Tenant, status: Tenant['status']) => {
    if (status === 'suspended' && !(await confirm({ title: `Suspend ${t.name}?`, message: 'Its employees can no longer sign in and its calls are not handled until it is activated again.', danger: true, confirmLabel: 'Suspend' }))) return;
    try {
      const updated = await put<Tenant>(`/platform/tenants/${t.id}`, { name: t.name, plan: t.plan, status });
      setData((l) => l?.map((x) => (x.id === t.id ? updated : x)));
      toast.success(`${t.name} ${status === 'active' ? 'activated' : 'suspended'}`);
    } catch (e) { toast.error(e); }
  };
  return (
    <>
      <PageHeader title="Organizations" sub="Platform administration — every organization is fully isolated"
        actions={<button className="btn primary" onClick={() => setCreating(true)}><Plus />New organization</button>} />
      <Card flush>
        {error ? <div style={{ padding: 16 }}><ErrorBox error={error} onRetry={reload} /></div>
          : loading && !data ? <SkeletonRows rows={4} />
          : !data?.length ? <Empty title="No organizations" icon={<Globe />} />
          : (
            <div className="table-wrap">
              <table className="table">
                <thead><tr><th>Organization</th><th>Slug</th><th>Plan</th><th>Status</th><th className="hide-sm">Created</th><th /></tr></thead>
                <tbody>
                  {data.map((t) => (
                    <tr key={t.id}>
                      <td><span className="strong" dir="auto">{t.name}</span>{t.id === current?.id && <span className="badge info" style={{ marginInlineStart: 6 }}>Current</span>}</td>
                      <td className="mono">{t.slug}</td>
                      <td>{t.plan || '—'}</td>
                      <td><span className={`badge ${t.status === 'active' ? 'ok' : 'danger'}`}>{t.status === 'active' ? 'Active' : 'Suspended'}</span></td>
                      <td className="hide-sm">{fmtDateTime(t.created_at)}</td>
                      <td className="actions">
                        {t.status === 'active'
                          ? <button className="btn sm" disabled={t.id === current?.id} title={t.id === current?.id ? 'You cannot suspend your own organization' : undefined} onClick={() => setStatus(t, 'suspended')}><Pause />Suspend</button>
                          : <button className="btn sm success" onClick={() => setStatus(t, 'active')}><Play />Activate</button>}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
      </Card>
      <CreateTenantDialog open={creating} onClose={() => setCreating(false)} onCreated={(t) => setData((l) => [...(l ?? []), t])} />
    </>
  );
}
