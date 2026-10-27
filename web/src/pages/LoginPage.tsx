import { useState, type FormEvent } from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { LogIn, Phone } from 'lucide-react';
import { useAuth } from '../state/auth';
import { errMsg } from '../api/client';
import { unlockAudio } from '../lib/ringtone';

export default function LoginPage() {
  const { user, login } = useAuth();
  const loc = useLocation();
  const from = (loc.state as { from?: string } | null)?.from;
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  // After a successful login the user is set and this redirects (Home picks the landing page by role).
  if (user) return <Navigate to={from && from !== '/login' ? from : '/'} replace />;

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    unlockAudio();
    setBusy(true);
    setError(null);
    try {
      await login(username.trim(), password);
    } catch (err) {
      setError(errMsg(err));
    } finally {
      setBusy(false);
    }
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
        </div>
      </form>
    </div>
  );
}
