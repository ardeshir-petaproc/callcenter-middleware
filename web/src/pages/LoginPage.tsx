import { useEffect, useState, type FormEvent } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { LogIn, Phone } from 'lucide-react';
import { useAuth } from '../state/auth';
import { errMsg } from '../api/client';
import type { ChooseTenant } from '../api/types_org';
import { unlockAudio } from '../lib/ringtone';
import { useLocalStorage } from '../lib/hooks';
import { JoowSignIn, urlJoowToken } from './auth/JoowSignIn';
import { TenantPicker } from './auth/TenantPicker';

export default function LoginPage() {
  const { user, login, loginJoow } = useAuth();
  const loc = useLocation();
  const from = (loc.state as { from?: string } | null)?.from;
  const [org, setOrg] = useLocalStorage('cc.login.org', 'default');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [choose, setChoose] = useState<{ token: string; res: ChooseTenant } | null>(null);

  const run = async (fn: () => Promise<void>) => {
    unlockAudio();
    setBusy(true);
    setError(null);
    try { await fn(); } catch (err) { setError(errMsg(err)); } finally { setBusy(false); }
  };
  const withJoow = (token: string, tenantId?: number) => run(async () => {
    const res = await loginJoow(token, tenantId);
    setChoose(res ? { token, res } : null);
  });

  // Opened from the Joow app with a session token.
  useEffect(() => {
    const t = urlJoowToken();
    if (t && !user) void withJoow(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // After a successful login the user is set and this redirects (Home picks the landing page).
  if (user) return <Navigate to={from && from !== '/login' ? from : '/'} replace />;

  const submit = (e: FormEvent) => {
    e.preventDefault();
    void run(async () => { await login(username.trim(), password, org.trim() || 'default'); });
  };

  return (
    <div className="login-page">
      <form className="card login-card" onSubmit={submit}>
        <div className="card-body">
          <div className="row" style={{ gap: 12 }}>
            <span className="brand-logo" style={{ width: 40, height: 40 }}><Phone /></span>
            <div>
              <h1>Call Center</h1>
              <div className="muted small">Sign in to the control panel</div>
            </div>
          </div>
          {error && <div className="alert danger" role="alert">{error}</div>}
          {choose ? (
            <>
              <TenantPicker tenants={choose.res.tenants} name={choose.res.account?.name} busy={busy} onPick={(t) => void withJoow(choose.token, t.id)} />
              <button type="button" className="btn ghost" onClick={() => setChoose(null)}>Back to sign in</button>
            </>
          ) : (
            <>
              <label className="field">
                <span className="field-label">Organization</span>
                <input className="input mono" autoComplete="organization" value={org} onChange={(e) => setOrg(e.target.value.toLowerCase())} />
              </label>
              <label className="field">
                <span className="field-label">Username</span>
                <input className="input" autoFocus autoComplete="username" value={username} onChange={(e) => setUsername(e.target.value)} required />
              </label>
              <label className="field">
                <span className="field-label">Password</span>
                <input className="input" type="password" autoComplete="current-password" value={password} onChange={(e) => setPassword(e.target.value)} required />
              </label>
              <button className="btn primary lg" type="submit" disabled={busy || !username || !password}>
                {busy ? <span className="spinner" /> : <LogIn />} Sign in
              </button>
              <div className="divider-text">or</div>
              <JoowSignIn busy={busy} onToken={(t) => withJoow(t)} />
            </>
          )}
        </div>
      </form>
    </div>
  );
}
