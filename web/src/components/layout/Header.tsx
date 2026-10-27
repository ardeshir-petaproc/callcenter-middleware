import { useLocation, useNavigate } from 'react-router-dom';
import { Menu, Moon, Sun, RefreshCw } from 'lucide-react';
import { titleFor } from './nav';
import { UserMenu } from './UserMenu';
import { useStore, useNow } from '../../lib/store';
import { live } from '../../state/live';
import { reconnectLive } from '../../state/ws';
import { useMyAgent } from '../../state/selectors';
import { cycleTheme, effectiveTheme, useTheme } from '../../lib/theme';
import { agentStatusLabel } from '../ui/Badges';
import { fmtDuration, secsSince } from '../../lib/format';

const connLabel = { idle: 'Offline', connecting: 'Connecting…', open: 'Live', closed: 'Reconnecting…' } as const;

function ConnIndicator() {
  const conn = useStore(live, (s) => s.conn);
  return (
    <button
      className={`btn ghost sm conn ${conn}`}
      onClick={() => conn !== 'open' && reconnectLive()}
      title={conn === 'open' ? 'Realtime connection active' : 'Click to reconnect now'}
      aria-label={`Realtime connection: ${connLabel[conn]}`}
    >
      <span className="dot" />
      <span className="hide-sm">{connLabel[conn]}</span>
      {conn === 'closed' && <RefreshCw size={13} />}
    </button>
  );
}

function MyStatusPill() {
  const me = useMyAgent();
  const now = useNow();
  const nav = useNavigate();
  if (!me) return null;
  return (
    <button className={`badge st st-${me.status}`} style={{ border: 0, cursor: 'pointer', height: 28, padding: '0 10px' }}
      onClick={() => nav('/console')} title="My status — open agent console">
      <span className="dot" />
      <span>{agentStatusLabel(me.status)}</span>
      <span className="num hide-sm" style={{ opacity: 0.8 }}>{fmtDuration(secsSince(me.since, now))}</span>
    </button>
  );
}

export function Header({ onMenu }: { onMenu: () => void }) {
  const loc = useLocation();
  useTheme();
  const dark = effectiveTheme() === 'dark';
  return (
    <header className="header">
      <button className="btn ghost icon mobile-only" onClick={onMenu} aria-label="Open navigation"><Menu /></button>
      <div className="title ellipsis grow">{titleFor(loc.pathname)}</div>
      <ConnIndicator />
      <MyStatusPill />
      <button className="btn ghost icon" onClick={cycleTheme} aria-label={dark ? 'Switch to light theme' : 'Switch to dark theme'} title="Toggle theme">
        {dark ? <Sun /> : <Moon />}
      </button>
      <UserMenu />
    </header>
  );
}
