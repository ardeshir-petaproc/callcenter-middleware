import { useState } from 'react';
import { MailPlus } from 'lucide-react';
import { CopyField } from '../../components/ui/CopyField';
import { post } from '../../api/client';
import type { Role } from '../../api/types';
import type { Invitation, InviteCreated } from '../../api/types_org';
import { useLookup } from '../../state/lookups';
import { Dialog } from '../../components/ui/Dialog';
import { Field } from '../../components/ui/primitives';
import { ChipSelect } from '../../components/ui/TagInput';
import { toast } from '../../components/ui/Toast';

const blank = { name: '', email: '', phone: '', joow_account_id: '', role: 'agent' as Role, custom_role_id: 0, department_ids: [] as string[] };

/** Creates an invitation and shows the one-time link to share. */
export function InviteDialog({ open, onClose, onCreated }: { open: boolean; onClose: () => void; onCreated: (i: Invitation) => void }) {
  const [f, setF] = useState(blank);
  const [created, setCreated] = useState<InviteCreated | null>(null);
  const [busy, setBusy] = useState(false);
  const roles = useLookup('customRoles');
  const depts = useLookup('departments');
  const set = (p: Partial<typeof blank>) => setF((x) => ({ ...x, ...p }));
  const close = () => { setF(blank); setCreated(null); onClose(); };
  const link = created ? inviteLink(created) : '';

  const submit = async () => {
    setBusy(true);
    try {
      const res = await post<InviteCreated>('/invitations', { ...f, custom_role_id: f.custom_role_id || undefined, department_ids: f.department_ids.map(Number) });
      setCreated(res);
      onCreated(res.invitation);
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };

  return (
    <Dialog open={open} onClose={close} title={created ? 'Invitation created' : 'Invite someone'}
      footer={created ? <button className="btn primary" onClick={close}>Done</button>
        : <><button className="btn" onClick={close}>Cancel</button><button className="btn primary" disabled={busy || !(f.email || f.phone || f.joow_account_id)} onClick={submit}><MailPlus />Create invitation</button></>}>
      {created ? (
        <>
          <p>Share this link with <strong dir="auto">{created.invitation.name || created.invitation.email || 'the invitee'}</strong>. It is shown only once and expires in 7 days.{created.invitation.joow_account_id ? ' Joow also notified them in-app.' : ''}</p>
          <CopyField value={link} label="Invitation link" />
        </>
      ) : (
        <div className="form-grid">
          <Field label="Name"><input className="input" dir="auto" value={f.name} onChange={(e) => set({ name: e.target.value })} data-autofocus /></Field>
          <Field label="Email"><input className="input" type="email" value={f.email} onChange={(e) => set({ email: e.target.value })} /></Field>
          <Field label="Phone"><input className="input" inputMode="tel" value={f.phone} onChange={(e) => set({ phone: e.target.value })} /></Field>
          <Field label="Joow account ID" hint="Optional: only this Joow account can accept"><input className="input mono" value={f.joow_account_id} onChange={(e) => set({ joow_account_id: e.target.value.trim() })} /></Field>
          <Field label="Role">
            <select className="select" value={f.role} onChange={(e) => set({ role: e.target.value as Role })}>
              <option value="agent">Agent</option><option value="supervisor">Supervisor</option><option value="admin">Admin</option>
            </select>
          </Field>
          <Field label="Custom role">
            <select className="select" value={f.custom_role_id || ''} onChange={(e) => set({ custom_role_id: Number(e.target.value) || 0 })}>
              <option value="">Built-in role</option>
              {roles.map((r) => <option key={r.id} value={r.id}>{r.name}</option>)}
            </select>
          </Field>
          {depts.length > 0 && <div className="field span-all"><span className="field-label">Departments</span>
            <ChipSelect options={depts.map((d) => ({ value: String(d.id), label: d.name }))} value={f.department_ids} onChange={(v) => set({ department_ids: v })} /></div>}
          <span className="hint span-all">An email, phone or Joow account is required.</span>
        </div>
      )}
    </Dialog>
  );
}

/** The server builds the link from PUBLIC_URL (the SPA serves both /accept-invite?token= and /invite/:token). */
function inviteLink(c: InviteCreated): string {
  return c.link || `${window.location.origin}/invite/${c.token}`;
}
