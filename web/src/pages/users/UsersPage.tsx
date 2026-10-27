import { useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { UsersRound } from 'lucide-react';
import type { QueueMember, User } from '../../api/types';
import { get } from '../../api/client';
import { useCrud } from '../../lib/crud';
import { useApi, useDraft } from '../../lib/hooks';
import { useMe } from '../../state/auth';
import { isAdmin } from '../../lib/roles';
import { useStore } from '../../lib/store';
import { live } from '../../state/live';
import { ConfigLayout, EditorFooter } from '../../components/ConfigLayout';
import { Card, Empty } from '../../components/ui/primitives';
import { Avatar, RoleBadge } from '../../components/ui/Badges';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';
import { blankUser, UserForm, type UserDraft } from './UserForm';
import { ForceStatus } from './ForceStatus';

export default function UsersPage() {
  const { id } = useParams();
  const nav = useNavigate();
  const me = useMe();
  const canEdit = isAdmin(me.role);
  const crud = useCrud<User>('/users', 'users');
  const members = useApi((s) => get<QueueMember[]>('/members', undefined, s), []);
  const agents = useStore(live, (s) => s.agents);
  const isNew = id === 'new';

  const source = useMemo<UserDraft | undefined>(() => {
    if (isNew) return { ...blankUser };
    const u = crud.items.find((x) => String(x.id) === id);
    if (!u) return undefined;
    const queues = (members.data ?? []).filter((m) => m.user_id === u.id).map(({ queue_id, penalty }) => ({ queue_id, penalty }));
    return { ...u, skills: u.skills ?? [], password: '', queues };
  }, [isNew, id, crud.items, members.data]);
  const { draft, set, dirty } = useDraft<UserDraft>(source, id ?? '');
  const [busy, setBusy] = useState(false);

  const save = async () => {
    if (!draft) return;
    if (!draft.username.trim()) { toast.error('Username is required'); return; }
    if (isNew && draft.password.length < 6) { toast.error('Password must be at least 6 characters'); return; }
    setBusy(true);
    try {
      const { password, queues, created_at: _c, updated_at: _u, ...rest } = draft;
      void _c; void _u;
      const body: Record<string, unknown> = { ...rest, queues };
      if (password) body.password = password;
      if (isNew) delete body.id;
      const saved = await crud.save(body as Partial<User>);
      members.reload();
      toast.success(isNew ? 'User created' : 'User saved');
      if (isNew) nav(`/users/${saved.id}`, { replace: true });
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  const remove = async () => {
    if (!draft) return;
    if (draft.id === me.id) { toast.error('You cannot delete your own account'); return; }
    if (!(await confirm({ title: 'Delete user?', message: `Delete ${draft.display_name || draft.username}? This cannot be undone.`, danger: true, confirmLabel: 'Delete' }))) return;
    try { await crud.remove(draft.id); toast.success('User deleted'); nav('/users', { replace: true }); } catch (e) { toast.error(e); }
  };

  return (
    <ConfigLayout
      title="Users & agents" sub="Accounts, roles, extensions and queue membership" basePath="/users"
      items={crud.items} loading={crud.loading} error={crud.error} onRetry={crud.reload}
      selected={id} canCreate={canEdit} newLabel="New user" emptyIcon={<UsersRound />}
      search={(u) => `${u.username} ${u.display_name} ${u.extension}`}
      renderItem={(u) => (
        <>
          <Avatar name={u.display_name || u.username} status={agents[u.id]?.status ?? 'offline'} />
          <div className="grow" style={{ minWidth: 0 }}>
            <div className="strong ellipsis" style={u.enabled ? undefined : { textDecoration: 'line-through' }}>{u.display_name || u.username}</div>
            <div className="small muted ellipsis">@{u.username}{u.extension ? ` · ${u.extension}` : ''}</div>
          </div>
          {u.role !== 'agent' && <RoleBadge role={u.role} />}
        </>
      )}
    >
      {!id ? <div className="card"><Empty title="Select a user" icon={<UsersRound />}>Choose someone on the left{canEdit ? ' or create a new account' : ''}.</Empty></div>
        : !draft ? <div className="card"><Empty title={crud.loading ? 'Loading…' : 'User not found'} /></div>
        : (
          <>
            {!isNew && <ForceStatus userId={draft.id} agent={agents[draft.id]} />}
            <Card title={isNew ? 'New user' : draft.display_name || draft.username} flush>
              <div className="card-body"><UserForm u={draft} set={set} disabled={!canEdit} isNew={isNew} /></div>
              <EditorFooter onSave={save} onDelete={remove} busy={busy} canEdit={canEdit} dirty={dirty} isNew={isNew} />
            </Card>
          </>
        )}
    </ConfigLayout>
  );
}
