import { useState } from 'react';
import { post } from '../../api/client';
import type { Tenant } from '../../api/types';
import type { TenantCreated } from '../../api/types_org';
import { Dialog } from '../../components/ui/Dialog';
import { Field, Segmented } from '../../components/ui/primitives';
import { toast } from '../../components/ui/Toast';

const blank = { slug: '', name: '', plan: '', username: '', password: '', display_name: '', email: '', joow_account_id: '' };
type OwnerKind = 'local' | 'joow';

/** Provisions an organization with its owner (local admin or an existing Joow account). */
export function CreateTenantDialog({ open, onClose, onCreated }: { open: boolean; onClose: () => void; onCreated: (t: Tenant) => void }) {
  const [f, setF] = useState(blank);
  const [kind, setKind] = useState<OwnerKind>('local');
  const [busy, setBusy] = useState(false);
  const set = (p: Partial<typeof blank>) => setF((x) => ({ ...x, ...p }));
  const close = () => { setF(blank); onClose(); };
  const slugOk = /^[a-z0-9][a-z0-9-]{1,40}$/.test(f.slug);
  const ownerOk = kind === 'joow' ? !!f.joow_account_id.trim() : !!f.username.trim() && f.password.length >= 8;
  const submit = async () => {
    setBusy(true);
    try {
      const owner = kind === 'joow'
        ? { joow_account_id: f.joow_account_id.trim(), display_name: f.display_name, email: f.email }
        : { username: f.username.trim(), password: f.password, display_name: f.display_name, email: f.email };
      const res = await post<TenantCreated>('/platform/tenants', { slug: f.slug, name: f.name.trim(), plan: f.plan, owner });
      toast.success(`${res.tenant.name} created — the owner signs in with organization “${res.tenant.slug}”`);
      onCreated(res.tenant);
      close();
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  return (
    <Dialog open={open} onClose={close} title="New organization"
      footer={<><button className="btn" onClick={close}>Cancel</button><button className="btn primary" disabled={busy || !slugOk || !f.name.trim() || !ownerOk} onClick={submit}>Create organization</button></>}>
      <div className="form-grid">
        <Field label="Name"><input className="input" dir="auto" value={f.name} onChange={(e) => set({ name: e.target.value })} data-autofocus /></Field>
        <Field label="Slug" hint={f.slug && !slugOk ? 'a-z, 0-9 and -, 2–41 characters' : 'Used to sign in'}>
          <input className="input mono" value={f.slug} onChange={(e) => set({ slug: e.target.value.toLowerCase() })} />
        </Field>
        <Field label="Plan"><input className="input" placeholder="e.g. pro" value={f.plan} onChange={(e) => set({ plan: e.target.value })} /></Field>
      </div>
      <div className="form-section">
        <h3>Owner (administrator)</h3>
        <Segmented<OwnerKind> label="Owner sign-in" value={kind} onChange={setKind} options={[{ value: 'local', label: 'Username & password' }, { value: 'joow', label: 'Joow account' }]} />
        <div className="form-grid">
          {kind === 'joow'
            ? <Field label="Joow account ID"><input className="input mono" value={f.joow_account_id} onChange={(e) => set({ joow_account_id: e.target.value })} /></Field>
            : <>
              <Field label="Username"><input className="input" autoComplete="off" value={f.username} onChange={(e) => set({ username: e.target.value })} /></Field>
              <Field label="Password" hint="At least 8 characters"><input className="input" type="password" autoComplete="new-password" value={f.password} onChange={(e) => set({ password: e.target.value })} /></Field>
            </>}
          <Field label="Display name"><input className="input" dir="auto" value={f.display_name} onChange={(e) => set({ display_name: e.target.value })} /></Field>
          <Field label="Email"><input className="input" type="email" value={f.email} onChange={(e) => set({ email: e.target.value })} /></Field>
        </div>
      </div>
    </Dialog>
  );
}
