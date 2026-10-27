import {
  Activity, BarChart3, Bot, CalendarClock, Headset, History, LayoutDashboard, ListTree, PhoneForwarded,
  Megaphone, Settings, UsersRound, Voicemail, Contact, Workflow, type LucideIcon,
} from 'lucide-react';
import type { Role } from '../../api/types';

export interface NavItem { to: string; label: string; icon: LucideIcon; min: Role; badge?: 'voicemail' }
export interface NavSection { title: string; items: NavItem[] }

export const NAV: NavSection[] = [
  {
    title: 'Operations',
    items: [
      { to: '/console', label: 'Agent console', icon: Headset, min: 'agent' },
      { to: '/dashboard', label: 'Dashboard', icon: LayoutDashboard, min: 'supervisor' },
      { to: '/live', label: 'Live calls', icon: Activity, min: 'supervisor' },
      { to: '/calls', label: 'Call history', icon: History, min: 'agent' },
      { to: '/callbacks', label: 'Callbacks', icon: PhoneForwarded, min: 'agent' },
      { to: '/voicemail', label: 'Voicemail', icon: Voicemail, min: 'agent', badge: 'voicemail' },
      { to: '/contacts', label: 'Contacts', icon: Contact, min: 'agent' },
      { to: '/campaigns', label: 'Campaigns', icon: Megaphone, min: 'supervisor' },
      { to: '/reports', label: 'Reports', icon: BarChart3, min: 'supervisor' },
    ],
  },
  {
    title: 'Configuration',
    items: [
      { to: '/queues', label: 'Queues', icon: ListTree, min: 'agent' },
      { to: '/users', label: 'Users & agents', icon: UsersRound, min: 'supervisor' },
      { to: '/flows', label: 'IVR flows', icon: Workflow, min: 'admin' },
      { to: '/ai', label: 'AI agents', icon: Bot, min: 'supervisor' },
      { to: '/schedules', label: 'Schedules', icon: CalendarClock, min: 'admin' },
      { to: '/settings', label: 'Settings', icon: Settings, min: 'admin' },
    ],
  },
];

export const ALL_NAV = NAV.flatMap((s) => s.items);

export function titleFor(path: string): string {
  const hit = ALL_NAV.filter((n) => path === n.to || path.startsWith(`${n.to}/`)).sort((a, b) => b.to.length - a.to.length)[0];
  return hit?.label ?? '';
}
