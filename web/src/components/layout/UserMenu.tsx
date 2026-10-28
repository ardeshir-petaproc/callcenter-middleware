import { useEffect, useRef, useState } from 'react';
import { Link } from 'react-router-dom';
import { Building2, KeyRound, LogOut, Monitor, Tv } from 'lucide-react';
import { useAuth, useMe } from '../../state/auth';
import { Avatar, RoleBadge } from '../ui/Badges';
import { Dialog } from '../ui/Dialog';
import { Field } from '../ui/primitives';
import { toast } from '../ui/Toast';
import { post } from '../../api/client';
import { PERMS } from '../../lib/perms';
import { setTheme, useTheme } from '../../lib/theme';
import { OrgSwitcher } from './OrgSwitcher';

export function UserMenu() {
  const me = useMe();
  const { logout, can, tenant } = useAuth();
  const [open, setOpen] = useState(false);
  const [pwOpen, setPwOpen] = useState(false);
  const [theme] = useTheme();
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const close = (e: MouseEvent) => { if (!ref.current?.contains(e.target as Node)) setOpen(false); };
    const esc = (e: KeyboardEvent) => { if (e.key === 'Escape') setOpen(false); };
    document.addEventListener('mousedown', close);
    document.addEventListener('keydown', esc);
    return () => { document.removeEventListener('mousedown', close); document.removeEventListener('keydown', esc); };
  }, [open]);

  return (
    <div className="menu-anchor" ref={ref}>
      <button className="btn ghost" style={{ padding: '0 4px' }} onClick={() => setOpen((o) => !o)} aria-haspopup="menu" aria-expanded={open} aria-label="User menu">
        <Avatar name={me.display_name || me.username} />
      </button>
      {open && (
        <div className="menu" role="menu">
          <div className="menu-head">
            <div className="strong">{me.display_name || me.username}</div>
            <div className="row small muted" style={{ marginTop: 4 }}>@{me.username} <RoleBadge role={me.role} /></div>
            {tenant && <div className="row small muted" style={{ marginTop: 4 }}><Building2 size={13} /><span dir="auto">{tenant.name}</span></div>}
          </div>
          <OrgSwitcher onDone={() => setOpen(false)} />
          {can(PERMS.callsAll) && (
            <Link to="/wallboard" role="menuitem" onClick={() => setOpen(false)}><Tv />Wallboard mode</Link>
          )}
          <button role="menuitem" onClick={() => { setOpen(false); setPwOpen(true); }}><KeyRound />Change password</button>
          <button role="menuitem" onClick={() => setTheme(theme === 'system' ? 'light' : 'system')}>
            <Monitor />{theme === 'system' ? 'Theme: follows system' : 'Theme: manual (reset)'}
          </button>
          <button role="menuitem" onClick={logout}><LogOut />Sign out</button>
        </div>
      )}
      <PasswordDialog open={pwOpen} onClose={() => setPwOpen(false)} />
    </div>
  );
}

function PasswordDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const [cur, setCur] = useState('');
  const [next, setNext] = useState('');
  const [confirm, setConfirm] = useState('');
  const [busy, setBusy] = useState(false);
  const mismatch = confirm.length > 0 && next !== confirm;
  const submit = async () => {
    setBusy(true);
    try {
      await post('/auth/password', { current_password: cur, new_password: next });
      toast.success('Password changed');
      setCur(''); setNext(''); setConfirm('');
      onClose();
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  return (
    <Dialog open={open} onClose={onClose} title="Change password" size="narrow"
      footer={<><button className="btn" onClick={onClose}>Cancel</button>
        <button className="btn primary" disabled={busy || !cur || next.length < 8 || next !== confirm} onClick={submit}>Update password</button></>}>
      <Field label="Current password"><input className="input" type="password" autoComplete="current-password" value={cur} onChange={(e) => setCur(e.target.value)} /></Field>
      <Field label="New password" hint="At least 8 characters"><input className="input" type="password" autoComplete="new-password" value={next} onChange={(e) => setNext(e.target.value)} /></Field>
      <Field label="Confirm new password" hint={mismatch ? 'Passwords do not match' : undefined}>
        <input className="input" type="password" autoComplete="new-password" value={confirm} onChange={(e) => setConfirm(e.target.value)} />
      </Field>
    </Dialog>
  );
}
