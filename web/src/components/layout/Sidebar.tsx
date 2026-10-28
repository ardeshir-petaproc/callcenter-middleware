import { NavLink } from 'react-router-dom';
import { PanelLeftClose, PanelLeftOpen, Phone } from 'lucide-react';
import { NAV } from './nav';
import { hasAccess } from '../../lib/access';
import { useAuth, useMe } from '../../state/auth';
import { useStore } from '../../lib/store';
import { live } from '../../state/live';

interface Props { collapsed: boolean; onToggle: () => void; onNavigate: () => void }

export function Sidebar({ collapsed, onToggle, onNavigate }: Props) {
  const me = useMe();
  const { can } = useAuth();
  const unheard = useStore(live, (s) => s.unheardVoicemails);
  return (
    <aside className="sidebar" aria-label="Main navigation">
      <div className="brand">
        <span className="brand-logo"><Phone /></span>
        <span>Call Center</span>
      </div>
      <nav className="nav">
        {NAV.map((section) => {
          const items = section.items.filter((i) => hasAccess(i, me, can));
          if (!items.length) return null;
          return (
            <div key={section.title} className="col" style={{ gap: 1 }}>
              <div className="nav-section">{section.title}</div>
              {items.map((item) => (
                <NavLink key={item.to} to={item.to} title={collapsed ? item.label : undefined} onClick={onNavigate}>
                  <item.icon aria-hidden />
                  <span>{item.label}</span>
                  {item.badge === 'voicemail' && unheard > 0 && <em className="count" aria-label={`${unheard} unheard`}>{unheard > 99 ? '99+' : unheard}</em>}
                </NavLink>
              ))}
            </div>
          );
        })}
      </nav>
      <div className="sidebar-foot desktop-only">
        <button className="btn ghost sm" style={{ width: '100%', justifyContent: collapsed ? 'center' : 'flex-start' }} onClick={onToggle} aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}>
          {collapsed ? <PanelLeftOpen /> : <><PanelLeftClose /><span>Collapse</span></>}
        </button>
      </div>
    </aside>
  );
}
