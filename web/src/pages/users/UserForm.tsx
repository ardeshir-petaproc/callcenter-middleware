import { Plus, Trash2 } from 'lucide-react';
import { useState } from 'react';
import type { Role, User } from '../../api/types';
import { Field, Toggle } from '../../components/ui/primitives';
import { TagInput } from '../../components/ui/TagInput';
import { useLookup } from '../../state/lookups';
import { UserAccessFields, UserProfileFields } from './UserProfileFields';

export type Membership = { queue_id: number; penalty: number };
export type UserDraft = User & { password: string; queues: Membership[] };

export const blankUser: UserDraft = {
  id: 0, username: '', display_name: '', email: '', role: 'agent', extension: '', endpoint: '', skills: [], enabled: true,
  created_at: '', updated_at: '', password: '', queues: [], status: 'active', joow_account_id: '', custom_role_id: 0,
  title: '', department_ids: [], manager_id: 0, languages: [], schedule_id: 0, job_profile_id: 0, phone: '',
};

interface Props { u: UserDraft; set: (p: Partial<UserDraft>) => void; disabled: boolean; isNew: boolean }

export function UserForm({ u, set, disabled, isNew }: Props) {
  const queues = useLookup('queues');
  const [pick, setPick] = useState('');
  const free = queues.filter((q) => !u.queues.some((m) => m.queue_id === q.id));
  return (
    <>
      <div className="form-section">
        <h3>Account</h3>
        <div className="form-grid">
          <Field label="Username"><input className="input" value={u.username} disabled={disabled || !isNew} autoComplete="off" onChange={(e) => set({ username: e.target.value })} /></Field>
          <Field label="Display name"><input className="input" dir="auto" value={u.display_name} disabled={disabled} onChange={(e) => set({ display_name: e.target.value })} /></Field>
          <Field label="Email"><input className="input" type="email" value={u.email} disabled={disabled} onChange={(e) => set({ email: e.target.value })} /></Field>
          <Field label="Role">
            <select className="select" value={u.role} disabled={disabled} onChange={(e) => set({ role: e.target.value as Role })}>
              <option value="agent">Agent</option><option value="supervisor">Supervisor</option><option value="admin">Admin</option>
            </select>
          </Field>
          <Field label={isNew ? 'Password' : 'New password'} hint={isNew ? (u.joow_account_id ? 'Optional: this employee signs in with Joow' : 'At least 6 characters') : 'Leave empty to keep the current password'}>
            <input className="input" type="password" autoComplete="new-password" value={u.password} disabled={disabled} onChange={(e) => set({ password: e.target.value })} />
          </Field>
          <div className="field" style={{ justifyContent: 'flex-end' }}>
            <Toggle checked={u.enabled} disabled={disabled} onChange={(v) => set({ enabled: v })} label="Account enabled" />
          </div>
        </div>
      </div>
      <UserAccessFields u={u} set={set} disabled={disabled} />
      <UserProfileFields u={u} set={set} disabled={disabled} />
      <div className="form-section">
        <h3>Telephony</h3>
        <div className="form-grid">
          <Field label="Extension" hint="Shown to callers and used for monitoring"><input className="input" value={u.extension} disabled={disabled} onChange={(e) => set({ extension: e.target.value })} /></Field>
          <Field label="Endpoint" hint="e.g. PJSIP/1001"><input className="input" value={u.endpoint} disabled={disabled} onChange={(e) => set({ endpoint: e.target.value })} /></Field>
          <div className="field span-all">
            <span className="field-label">Skills</span>
            <TagInput value={u.skills ?? []} onChange={(v) => set({ skills: v })} disabled={disabled} placeholder="e.g. spanish, billing" />
          </div>
        </div>
      </div>
      <div className="form-section">
        <h3>Queue memberships</h3>
        {u.queues.length === 0 && <div className="small muted">Not a member of any queue.</div>}
        {u.queues.map((m) => (
          <div key={m.queue_id} className="row">
            <span className="grow strong">{queues.find((q) => q.id === m.queue_id)?.name ?? `Queue #${m.queue_id}`}</span>
            <label className="row small muted">Penalty
              <input className="input sm" type="number" min={0} style={{ width: 80 }} value={m.penalty} disabled={disabled}
                onChange={(e) => set({ queues: u.queues.map((x) => (x.queue_id === m.queue_id ? { ...x, penalty: Number(e.target.value) } : x)) })} />
            </label>
            {!disabled && <button type="button" className="btn ghost icon sm" aria-label="Remove membership" onClick={() => set({ queues: u.queues.filter((x) => x.queue_id !== m.queue_id) })}><Trash2 /></button>}
          </div>
        ))}
        {!disabled && free.length > 0 && (
          <div className="row">
            <select className="select sm" value={pick} onChange={(e) => setPick(e.target.value)} aria-label="Add queue">
              <option value="">Add to queue…</option>
              {free.map((q) => <option key={q.id} value={q.id}>{q.name}</option>)}
            </select>
            <button type="button" className="btn sm" disabled={!pick} onClick={() => { set({ queues: [...u.queues, { queue_id: Number(pick), penalty: 0 }] }); setPick(''); }}><Plus />Add</button>
          </div>
        )}
      </div>
    </>
  );
}
