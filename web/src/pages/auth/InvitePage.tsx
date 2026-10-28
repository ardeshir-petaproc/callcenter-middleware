import { useState, type FormEvent } from 'react';
import { Link, Navigate, useParams, useSearchParams } from 'react-router-dom';
import { MailCheck, UserPlus } from 'lucide-react';
import { api, errMsg, get } from '../../api/client';
import type { LoginResponse } from '../../api/types';
import type { InvitePeek } from '../../api/types_org';
import { useApi } from '../../lib/hooks';
import { fmtDateTime } from '../../lib/format';
import { useAuth } from '../../state/auth';
import { RoleBadge } from '../../components/ui/Badges';
import { JoowSignIn } from './JoowSignIn';

/** Accepts an invitation (/invite/:token or /accept-invite?token=) with Joow or a new username/password. */
export default function InvitePage() {
  const params = useParams();
  const [search] = useSearchParams();
  const token = params.token ?? search.get('token') ?? '';
  const { user, adopt } = useAuth();
  const peek = useApi((s) => get<InvitePeek>(`/auth/invite/${encodeURIComponent(token)}`, undefined, s), [token], !!token);
  const [username, setUsername] = useState('');
  const [name, setName] = useState('');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [done, setDone] = useState(false);

  if (done && user) return <Navigate to="/" replace />;
  const inv = peek.data;
  const joowOnly = !!(inv?.joow_account_required || inv?.joow_account_id);

  const accept = async (body: Record<string, unknown>) => {
    setBusy(true);
    setError(null);
    try {
      const res = await api<LoginResponse>('/auth/accept-invite', { method: 'POST', body: { token, ...body }, noAuthRedirect: true });
      adopt(res);
      setDone(true);
    } catch (e) { setError(errMsg(e)); } finally { setBusy(false); }
  };
  const submit = (e: FormEvent) => {
    e.preventDefault();
    void accept({ username: username.trim(), password, display_name: name.trim() || inv?.name });
  };

  return (
    <div className="login-page">
      <form className="card login-card" onSubmit={submit}>
        <div className="card-body">
          <div className="row" style={{ gap: 12 }}>
            <span className="brand-logo" style={{ width: 40, height: 40 }}><MailCheck /></span>
            <div><h1>Join the team</h1><div className="muted small">Accept your invitation</div></div>
          </div>
          {!token || peek.error ? (
            <>
              <div className="alert danger" role="alert">{peek.error ?? 'Missing invitation token.'} Ask your manager for a new invitation.</div>
              <Link to="/login" className="btn">Go to sign in</Link>
            </>
          ) : !inv ? <span className="spinner" /> : (
            <>
              <div className="col" style={{ gap: 4 }}>
                <div>You were invited to <strong dir="auto">{inv.organization}</strong>{inv.name ? <> as <strong dir="auto">{inv.name}</strong></> : null}.</div>
                <div className="row small muted">Role <RoleBadge role={inv.role} />{inv.expires_at && <span>· expires {fmtDateTime(inv.expires_at)}</span>}</div>
              </div>
              {error && <div className="alert danger" role="alert">{error}</div>}
              {user && <div className="alert warn">You are signed in as {user.display_name || user.username}; accepting starts a new session.</div>}
              <JoowSignIn busy={busy} label="Accept with Joow" onToken={(t) => accept({ joow_token: t })} />
              {joowOnly ? <div className="small muted">This invitation is linked to a Joow account: accept it by signing in with Joow.</div> : (
                <>
                  <div className="divider-text">or create a password</div>
                  <label className="field"><span className="field-label">Your name</span>
                    <input className="input" dir="auto" value={name} placeholder={inv.name} onChange={(e) => setName(e.target.value)} /></label>
                  <label className="field"><span className="field-label">Username</span>
                    <input className="input" autoComplete="username" value={username} onChange={(e) => setUsername(e.target.value)} required /></label>
                  <label className="field"><span className="field-label">Password</span>
                    <input className="input" type="password" autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)} required />
                    <span className="hint">At least 8 characters</span></label>
                  <button className="btn primary lg" type="submit" disabled={busy || !username.trim() || password.length < 8}>
                    {busy ? <span className="spinner" /> : <UserPlus />} Create account
                  </button>
                </>
              )}
            </>
          )}
        </div>
      </form>
    </div>
  );
}
