import { useParams } from 'react-router-dom';
import { ShieldCheck } from 'lucide-react';
import { get } from '../../api/client';
import type { CustomRole, PermissionsResponse } from '../../api/types_org';
import { useCrud } from '../../lib/crud';
import { useApi } from '../../lib/hooks';
import { useEntityEditor } from '../../lib/editor';
import { ALL_PERMS, PERMS, PERM_LABELS } from '../../lib/perms';
import { useCan } from '../../state/auth';
import { ConfigLayout, EditorFooter } from '../../components/ConfigLayout';
import { RevisionsButton } from '../../components/RevisionsButton';
import { Card, Empty, Field } from '../../components/ui/primitives';

const blank: CustomRole = { id: 0, name: '', description: '', permissions: [] };

export default function RolesPage() {
  const { id } = useParams();
  const canEdit = useCan()(PERMS.usersManage);
  const crud = useCrud<CustomRole>('/roles', 'customRoles');
  const perms = useApi((s) => get<PermissionsResponse>('/permissions', undefined, s), []);
  const all = perms.data?.all ?? ALL_PERMS;
  const ed = useEntityEditor({
    crud, id, blank, base: '/org/roles', noun: 'Role', label: (r) => r.name,
    validate: (r) => (r.name.trim() ? null : 'Name is required'),
    normalize: (r) => ({ ...r, permissions: r.permissions ?? [] }),
  });
  const r = ed.draft;
  const toggle = (p: string, on: boolean) => r && ed.set({ permissions: on ? [...r.permissions, p] : r.permissions.filter((x) => x !== p) });

  return (
    <ConfigLayout
      title="Roles" sub="Custom roles grant an explicit set of permissions (least privilege)" basePath="/org/roles"
      items={crud.items} loading={crud.loading} error={crud.error} onRetry={crud.reload}
      selected={id} canCreate={canEdit} newLabel="New role" emptyIcon={<ShieldCheck />}
      search={(x) => `${x.name} ${x.description}`}
      renderItem={(x) => (
        <div className="grow">
          <div className="strong ellipsis" dir="auto">{x.name}</div>
          <div className="small muted">{(x.permissions ?? []).length} permission(s)</div>
        </div>
      )}
    >
      {!id ? (
        <div className="card"><Empty title="Select a role" icon={<ShieldCheck />}>
          Built-in roles: admin has every permission, supervisor has operations permissions, agent works on their own calls and tickets.
          A custom role assigned to an employee replaces the built-in role's permissions (admins always keep all).
        </Empty></div>
      ) : !r ? <div className="card"><Empty title={crud.loading ? 'Loading…' : 'Role not found'} /></div>
        : (
          <Card title={ed.isNew ? 'New role' : r.name || 'Role'} flush>
            <div className="card-body col gap-lg">
              <div className="form-grid">
                <Field label="Name"><input className="input" dir="auto" value={r.name} disabled={!canEdit} onChange={(e) => ed.set({ name: e.target.value })} /></Field>
                <Field label="Description" className="span-2"><input className="input" dir="auto" value={r.description} disabled={!canEdit} onChange={(e) => ed.set({ description: e.target.value })} /></Field>
              </div>
              <div className="form-section">
                <div className="row between">
                  <h3>Permissions ({r.permissions.length}/{all.length})</h3>
                  {canEdit && <div className="row"><button type="button" className="btn ghost sm" onClick={() => ed.set({ permissions: [...all] })}>Select all</button><button type="button" className="btn ghost sm" onClick={() => ed.set({ permissions: [] })}>Clear</button></div>}
                </div>
                <div className="check-grid">
                  {all.map((p) => (
                    <label key={p} className="checkbox check-card">
                      <input type="checkbox" checked={r.permissions.includes(p)} disabled={!canEdit} onChange={(e) => toggle(p, e.target.checked)} />
                      <span className="col" style={{ gap: 0 }}><code>{p}</code><span className="small muted">{PERM_LABELS[p] ?? ''}</span></span>
                    </label>
                  ))}
                </div>
              </div>
            </div>
            <EditorFooter onSave={ed.save} onDelete={ed.remove} busy={ed.busy} canEdit={canEdit} dirty={ed.dirty} isNew={ed.isNew}
              extra={!ed.isNew && <RevisionsButton entity="custom_role" id={r.id} />} />
          </Card>
        )}
    </ConfigLayout>
  );
}
