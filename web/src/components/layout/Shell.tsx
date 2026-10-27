import { useEffect, useState } from 'react';
import { Outlet, useLocation, useNavigate } from 'react-router-dom';
import { PhoneIncoming } from 'lucide-react';
import { Sidebar } from './Sidebar';
import { Header } from './Header';
import { useStore } from '../../lib/store';
import { live } from '../../state/live';
import { get } from '../../api/client';
import type { Voicemail } from '../../api/types';
import { notify, startRinging, stopRinging, unlockAudio } from '../../lib/ringtone';
import { useLocalStorage } from '../../lib/hooks';
import { ErrorBoundary } from '../ErrorBoundary';

/** Rings + notifies on screen-pops anywhere in the app, and offers a jump to the console. */
function ScreenPopWatcher() {
  const pop = useStore(live, (s) => s.screenpop);
  const loc = useLocation();
  const nav = useNavigate();
  const popId = pop?.call.id;
  useEffect(() => {
    if (!pop) { stopRinging(); return; }
    startRinging();
    const who = pop.call.caller_name || pop.call.caller_number || 'Unknown caller';
    notify('Incoming call', `${who}${pop.call.caller_name ? ` (${pop.call.caller_number})` : ''}`, () => nav('/console'));
    return () => stopRinging();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [popId]);
  if (!pop || loc.pathname === '/console') return null;
  return (
    <div className="toasts" style={{ left: 16, right: 'auto', bottom: 16 }}>
      <div className="toast screenpop" role="alert">
        <PhoneIncoming style={{ color: 'var(--info-solid)' }} />
        <div className="grow">
          <div className="strong">Incoming call</div>
          <div>{pop.call.caller_name || pop.call.caller_number}</div>
        </div>
        <button className="btn primary sm" onClick={() => nav('/console')}>Open console</button>
      </div>
    </div>
  );
}

export function Shell() {
  const loc = useLocation();
  const [collapsed, setCollapsed] = useLocalStorage('cc.sidebar.collapsed', false);
  const [mobileOpen, setMobileOpen] = useState(false);

  useEffect(() => {
    get<Voicemail[]>('/voicemails')
      .then((v) => live.set({ unheardVoicemails: (v ?? []).filter((x) => !x.heard).length }))
      .catch(() => undefined);
    // First interaction unlocks Web Audio so the ringtone can play later.
    const unlock = () => unlockAudio();
    window.addEventListener('pointerdown', unlock, { once: true });
    window.addEventListener('keydown', unlock, { once: true });
    return () => { window.removeEventListener('pointerdown', unlock); window.removeEventListener('keydown', unlock); };
  }, []);

  return (
    <div className={`shell ${collapsed ? 'collapsed' : ''} ${mobileOpen ? 'mobile-open' : ''}`}>
      <Sidebar collapsed={collapsed} onToggle={() => setCollapsed(!collapsed)} onNavigate={() => setMobileOpen(false)} />
      {mobileOpen && <div className="scrim" onClick={() => setMobileOpen(false)} />}
      <div className="main">
        <Header onMenu={() => setMobileOpen(true)} />
        <main className="content">
          <ErrorBoundary resetKey={loc.pathname}><Outlet /></ErrorBoundary>
        </main>
      </div>
      <ScreenPopWatcher />
    </div>
  );
}
