import { useState } from 'react';
import { KeyRound } from 'lucide-react';
import { api, ApiError } from '../../api/client';

/** Obtains a Joow session token. In local/dev Joow mode the server mints one for an account id. */
export async function devJoowToken(accountId: string): Promise<string> {
  try {
    const res = await api<{ token: string }>('/dev/joow-token', { method: 'POST', body: { account_id: accountId }, noAuthRedirect: true });
    return res.token;
  } catch (e) {
    if (e instanceof ApiError && e.status === 404) {
      throw new Error(e.message === 'unknown account' ? 'Unknown Joow account' : 'Joow sign-in from the browser is not available here — open the panel from the Joow app.');
    }
    throw e;
  }
}

/** Joow token from the URL (when opened from the Joow app), if any. */
export function urlJoowToken(): string | null {
  return new URLSearchParams(window.location.search).get('joow_token');
}

/** "Sign in with Joow" control; in development it asks for the Joow account to simulate. */
export function JoowSignIn({ onToken, busy, label = 'Sign in with Joow' }: { onToken: (token: string) => Promise<void>; busy: boolean; label?: string }) {
  const [open, setOpen] = useState(false);
  const [account, setAccount] = useState('');
  const [err, setErr] = useState<string | null>(null);
  const go = async () => {
    setErr(null);
    try { await onToken(await devJoowToken(account.trim())); } catch (e) { setErr((e as Error).message); }
  };
  if (!open) {
    return <button type="button" className="btn lg joow-btn" disabled={busy} onClick={() => setOpen(true)}><KeyRound />{label}</button>;
  }
  return (
    <div className="col" style={{ gap: 8 }}>
      <label className="field">
        <span className="field-label">Joow account</span>
        <input className="input mono" autoFocus placeholder="e.g. j-alice" value={account} onChange={(e) => setAccount(e.target.value)}
          onKeyDown={(e) => { if (e.key === 'Enter' && account.trim()) { e.preventDefault(); void go(); } }} />
        <span className="hint">Development sign-in: simulates the Joow app for this account.</span>
      </label>
      {err && <div className="alert danger" role="alert">{err}</div>}
      <div className="row">
        <button type="button" className="btn" onClick={() => setOpen(false)}>Back</button>
        <button type="button" className="btn primary grow" disabled={busy || !account.trim()} onClick={go}>{busy ? <span className="spinner" /> : <KeyRound />}Continue with Joow</button>
      </div>
    </div>
  );
}
