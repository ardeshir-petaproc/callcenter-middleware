import { useState } from 'react';
import { Ban, MailPlus } from 'lucide-react';
import { post } from '../../api/client';
import type { Invitation } from '../../api/types_org';
import { useCrud } from '../../lib/crud';
import { fmtDateTime, humanize } from '../../lib/format';
import { useDepartmentNames, useLookup } from '../../state/lookups';
import { Card, Empty, ErrorBox, SkeletonRows } from '../../components/ui/primitives';
import { RoleBadge } from '../../components/ui/Badges';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';
import { InviteDialog } from './InviteDialog';

const tone: Record<string, string> = { pending: 'warn', accepted: 'ok', revoked: '' };

export function InvitationsPanel({ canEdit }: { canEdit: boolean }) {
  const crud = useCrud<Invitation>('/invitations');
  const roles = useLookup('customRoles');
  const deptName = useDepartmentNames();
  const [open, setOpen] = useState(false);

  const revoke = async (i: Invitation) => {
    if (!(await confirm({ title: 'Revoke invitation?', message: `The link sent to ${i.name || i.email || i.phone} will stop working.`, danger: true, confirmLabel: 'Revoke' }))) return;
    try {
      const updated = await post<Invitation>(`/invitations/${i.id}/revoke`);
      crud.setData((l) => l?.map((x) => (x.id === i.id ? updated : x)));
      toast.success('Invitation revoked');
    } catch (e) { toast.error(e); }
  };
  const items = [...crud.items].sort((a, b) => b.id - a.id);

  return (
    <Card title="Invitations" icon={<MailPlus />} flush actions={canEdit && <button className="btn sm primary" onClick={() => setOpen(true)}><MailPlus />Invite someone</button>}>
      {crud.error ? <div style={{ padding: 16 }}><ErrorBox error={crud.error} onRetry={crud.reload} /></div>
        : crud.loading && !crud.data ? <SkeletonRows rows={4} />
        : items.length === 0 ? <Empty title="No invitations" icon={<MailPlus />}>Invite people who are not employees yet; they accept with Joow or a password.</Empty>
        : (
          <div className="table-wrap">
            <table className="table">
              <thead><tr><th>Person</th><th>Role</th><th className="hide-sm">Departments</th><th>Status</th><th className="hide-sm">Expires</th><th /></tr></thead>
              <tbody>
                {items.map((i) => (
                  <tr key={i.id}>
                    <td><div className="strong" dir="auto">{i.name || '—'}</div><div className="sub">{[i.email, i.phone, i.joow_account_id && `Joow ${i.joow_account_id}`].filter(Boolean).join(' · ')}</div></td>
                    <td><RoleBadge role={i.role} />{i.custom_role_id > 0 && <span className="chip" style={{ marginLeft: 4 }}>{roles.find((r) => r.id === i.custom_role_id)?.name ?? `role #${i.custom_role_id}`}</span>}</td>
                    <td className="hide-sm">{(i.department_ids ?? []).map(deptName).join(', ') || '—'}</td>
                    <td><span className={`badge ${tone[i.status] ?? ''}`}>{humanize(i.status)}</span></td>
                    <td className="hide-sm">{fmtDateTime(i.expires_at)}</td>
                    <td className="actions">{canEdit && i.status === 'pending' && <button className="btn sm" onClick={() => revoke(i)}><Ban />Revoke</button>}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      <InviteDialog open={open} onClose={() => setOpen(false)} onCreated={(inv) => crud.setData((l) => [...(l ?? []), inv])} />
    </Card>
  );
}
