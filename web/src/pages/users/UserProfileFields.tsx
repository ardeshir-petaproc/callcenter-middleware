import type { User, UserStatus } from '../../api/types';
import { Field } from '../../components/ui/primitives';
import { ChipSelect, TagInput } from '../../components/ui/TagInput';
import { useLookup } from '../../state/lookups';

type P = Pick<User, 'title' | 'department_ids' | 'manager_id' | 'languages' | 'schedule_id' | 'job_profile_id' | 'phone' | 'id'>;
interface Props<T extends P> { u: T; set: (p: Partial<T>) => void; disabled: boolean }

export const LANG_SUGGESTIONS = ['en', 'fa', 'ar', 'tr', 'de', 'fr', 'es'];

/** Organizational profile of an employee: departments, manager, languages, hours and job profile. */
export function UserProfileFields<T extends P>({ u, set, disabled }: Props<T>) {
  const depts = useLookup('departments');
  const users = useLookup('users');
  const schedules = useLookup('schedules');
  const jobs = useLookup('jobProfiles');
  const s = set as (p: Partial<P>) => void;
  return (
    <div className="form-section">
      <h3>Profile</h3>
      <div className="form-grid">
        <Field label="Job title"><input className="input" dir="auto" value={u.title ?? ''} disabled={disabled} onChange={(e) => s({ title: e.target.value })} /></Field>
        <Field label="Job profile" hint="Routing rules can target everyone holding a profile">
          <select className="select" value={u.job_profile_id || ''} disabled={disabled} onChange={(e) => s({ job_profile_id: Number(e.target.value) || 0 })}>
            <option value="">None</option>
            {jobs.map((j) => <option key={j.id} value={j.id}>{j.title}</option>)}
          </select>
        </Field>
        <Field label="Manager">
          <select className="select" value={u.manager_id || ''} disabled={disabled} onChange={(e) => s({ manager_id: Number(e.target.value) || 0 })}>
            <option value="">None</option>
            {users.filter((x) => x.id !== u.id).map((x) => <option key={x.id} value={x.id}>{x.display_name || x.username}</option>)}
          </select>
        </Field>
        <Field label="Working hours">
          <select className="select" value={u.schedule_id || ''} disabled={disabled} onChange={(e) => s({ schedule_id: Number(e.target.value) || 0 })}>
            <option value="">Not set</option>
            {schedules.map((x) => <option key={x.id} value={x.id}>{x.name}</option>)}
          </select>
        </Field>
        <Field label="Phone"><input className="input" inputMode="tel" value={u.phone ?? ''} disabled={disabled} onChange={(e) => s({ phone: e.target.value })} /></Field>
        <div className="field span-all">
          <span className="field-label">Departments</span>
          {depts.length ? <ChipSelect options={depts.map((d) => ({ value: String(d.id), label: d.name }))} disabled={disabled}
            value={(u.department_ids ?? []).map(String)} onChange={(v) => s({ department_ids: v.map(Number) })} />
            : <span className="small muted">No departments defined yet.</span>}
        </div>
        <div className="field span-all">
          <span className="field-label">Languages</span>
          <TagInput value={u.languages ?? []} onChange={(v) => s({ languages: v })} disabled={disabled} placeholder="e.g. en, fa" suggestions={LANG_SUGGESTIONS} />
        </div>
      </div>
    </div>
  );
}

/** Custom role, Joow link and account status. */
export function UserAccessFields({ u, set, disabled }: { u: User; set: (p: Partial<User>) => void; disabled: boolean }) {
  const roles = useLookup('customRoles');
  return (
    <div className="form-section">
      <h3>Access & identity</h3>
      <div className="form-grid">
        <Field label="Custom role" hint="Replaces the built-in role's permissions (admins keep all)">
          <select className="select" value={u.custom_role_id || ''} disabled={disabled} onChange={(e) => set({ custom_role_id: Number(e.target.value) || 0 })}>
            <option value="">Built-in role permissions</option>
            {roles.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
          </select>
        </Field>
        <Field label="Joow account ID" hint="Linked employees sign in with Joow">
          <input className="input mono" value={u.joow_account_id ?? ''} disabled={disabled} onChange={(e) => set({ joow_account_id: e.target.value.trim() })} />
        </Field>
        <Field label="Status">
          <select className="select" value={u.status || 'active'} disabled={disabled} onChange={(e) => set({ status: e.target.value as UserStatus })}>
            <option value="active">Active</option><option value="invited">Invited</option><option value="disabled">Disabled</option>
          </select>
        </Field>
      </div>
    </div>
  );
}
