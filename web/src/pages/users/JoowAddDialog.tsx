import { useState } from 'react';
import { Search, UserPlus } from 'lucide-react';
import { get, post } from '../../api/client';
import type { Role, User } from '../../api/types';
import type { JoowAccount } from '../../api/types_org';
import { useApi, useDebounced } from '../../lib/hooks';
import { refreshLookup, useLookup } from '../../state/lookups';
import { Dialog } from '../../components/ui/Dialog';
import { Empty, ErrorBox, Field, SkeletonRows } from '../../components/ui/primitives';
import { ChipSelect } from '../../components/ui/TagInput';
import { Avatar } from '../../components/ui/Badges';
import { toast } from '../../components/ui/Toast';

/** Search the Joow directory and create an employee linked to the chosen account (signs in with Joow). */
export function JoowAddDialog({ open, onClose, onCreated }: { open: boolean; onClose: () => void; onCreated: (u: User) => void }) {
  const [q, setQ] = useState('');
  const dq = useDebounced(q, 300);
  const [pick, setPick] = useState<JoowAccount | null>(null);
  const [role, setRole] = useState<Role>('agent');
  const [customRole, setCustomRole] = useState(0);
  const [depts, setDepts] = useState<string[]>([]);
  const [ext, setExt] = useState('');
  const [busy, setBusy] = useState(false);
  const roles = useLookup('customRoles');
  const departments = useLookup('departments');
  const users = useLookup('users');
  const results = useApi((s) => get<JoowAccount[]>('/joow/accounts', { q: dq }, s), [dq], open);
  const linked = new Set(users.map((u) => u.joow_account_id).filter(Boolean));

  const close = () => { setPick(null); setQ(''); onClose(); };
  const create = async () => {
    if (!pick) return;
    setBusy(true);
    try {
      const u = await post<User>('/users/from-joow', {
        joow_account_id: pick.id, role, custom_role_id: customRole || undefined, extension: ext || undefined,
        department_ids: depts.map(Number), phone: pick.phone,
      });
      void refreshLookup('users');
      toast.success(`${u.display_name || pick.name} added — they sign in with Joow`);
      onCreated(u);
      close();
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };

  return (
    <Dialog open={open} onClose={close} title="Add employee from Joow"
      footer={<><button className="btn" onClick={close}>Cancel</button><button className="btn primary" disabled={!pick || busy} onClick={create}><UserPlus />Add employee</button></>}>
      <div className="input-icon"><Search /><input className="input" dir="auto" placeholder="Search Joow by name, email or phone…" value={q} onChange={(e) => setQ(e.target.value)} data-autofocus /></div>
      <div className="list" style={{ border: '1px solid var(--border)', borderRadius: 8, maxHeight: 240, overflowY: 'auto' }}>
        {results.error ? <div style={{ padding: 10 }}><ErrorBox error={results.error} onRetry={results.reload} /></div>
          : results.loading && !results.data ? <SkeletonRows rows={3} />
          : !results.data?.length ? <Empty title="No Joow accounts found" />
          : results.data.map((a) => (
            <button key={a.id} type="button" className={`list-item clickable plain-button ${pick?.id === a.id ? 'selected' : ''}`}
              disabled={linked.has(a.id)} onClick={() => setPick(a)}>
              <Avatar name={a.name} />
              <div className="grow" style={{ textAlign: 'left' }}>
                <div className="strong" dir="auto">{a.name}</div>
                <div className="small muted">{[a.email, a.phone].filter(Boolean).join(' · ')}</div>
              </div>
              {linked.has(a.id) && <span className="badge">Already an employee</span>}
            </button>
          ))}
      </div>
      {pick && (
        <div className="form-grid">
          <Field label="Role">
            <select className="select" value={role} onChange={(e) => setRole(e.target.value as Role)}>
              <option value="agent">Agent</option><option value="supervisor">Supervisor</option><option value="admin">Admin</option>
            </select>
          </Field>
          <Field label="Custom role">
            <select className="select" value={customRole || ''} onChange={(e) => setCustomRole(Number(e.target.value) || 0)}>
              <option value="">Built-in role</option>
              {roles.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
            </select>
          </Field>
          <Field label="Extension"><input className="input" value={ext} onChange={(e) => setExt(e.target.value)} /></Field>
          {departments.length > 0 && (
            <div className="field span-all"><span className="field-label">Departments</span>
              <ChipSelect options={departments.map((d) => ({ value: String(d.id), label: d.name }))} value={depts} onChange={setDepts} /></div>
          )}
        </div>
      )}
    </Dialog>
  );
}
