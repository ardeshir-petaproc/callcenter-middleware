import {
  Activity, BarChart3, BookOpen, Bot, Building2, CalendarClock, ClipboardCheck, Contact, GitFork, Globe, Headset,
  History, IdCard, LayoutDashboard, ListTree, Megaphone, Phone, PhoneForwarded, Settings, ShieldCheck,
  Star, Ticket, UsersRound, Voicemail, Workflow, type LucideIcon,
} from 'lucide-react';
import type { Access } from '../../lib/access';
import { PERMS } from '../../lib/perms';

export interface NavItem extends Access { to: string; label: string; icon: LucideIcon; badge?: 'voicemail' }
export interface NavSection { title: string; items: NavItem[] }

export const NAV: NavSection[] = [
  {
    title: 'Operations',
    items: [
      { to: '/console', label: 'Agent console', icon: Headset, min: 'agent' },
      { to: '/dashboard', label: 'Dashboard', icon: LayoutDashboard, perm: PERMS.callsAll },
      { to: '/live', label: 'Live calls', icon: Activity, perm: PERMS.callsAll },
      { to: '/calls', label: 'Call history', icon: History, min: 'agent' },
      { to: '/tickets', label: 'Tickets', icon: Ticket, min: 'agent' },
      { to: '/callbacks', label: 'Callbacks', icon: PhoneForwarded, min: 'agent' },
      { to: '/voicemail', label: 'Voicemail', icon: Voicemail, min: 'agent', badge: 'voicemail' },
      { to: '/contacts', label: 'Contacts', icon: Contact, min: 'agent' },
      { to: '/kb', label: 'Knowledge base', icon: BookOpen, min: 'agent' },
      { to: '/campaigns', label: 'Campaigns', icon: Megaphone, perm: PERMS.campaignsManage },
      { to: '/reports', label: 'Reports', icon: BarChart3, perm: PERMS.reportsView },
    ],
  },
  {
    title: 'Quality',
    items: [
      { to: '/quality', label: 'Quality', icon: ClipboardCheck, perm: PERMS.qaEvaluate },
      { to: '/my-evaluations', label: 'My evaluations', icon: Star, min: 'agent' },
    ],
  },
  {
    title: 'Organization',
    items: [
      { to: '/users', label: 'Employees', icon: UsersRound, min: 'supervisor', perm: PERMS.usersManage },
      { to: '/org/departments', label: 'Departments', icon: Building2, perm: PERMS.orgManage },
      { to: '/org/job-profiles', label: 'Job profiles', icon: IdCard, perm: PERMS.orgManage },
      { to: '/org/roles', label: 'Roles', icon: ShieldCheck, perm: PERMS.usersManage },
      { to: '/routing', label: 'Routing', icon: GitFork, perm: [PERMS.routingManage, PERMS.routingApprove] },
    ],
  },
  {
    title: 'Configuration',
    items: [
      { to: '/queues', label: 'Queues', icon: ListTree, min: 'agent' },
      { to: '/flows', label: 'IVR flows', icon: Workflow, perm: PERMS.flowsManage },
      { to: '/org/numbers', label: 'Phone numbers', icon: Phone, perm: PERMS.flowsManage },
      { to: '/ai', label: 'AI agents', icon: Bot, min: 'supervisor', perm: PERMS.aiManage },
      { to: '/schedules', label: 'Schedules', icon: CalendarClock, perm: PERMS.queuesManage },
      { to: '/settings', label: 'Settings', icon: Settings, perm: PERMS.settingsManage },
    ],
  },
  {
    title: 'Platform',
    items: [{ to: '/platform', label: 'Organizations', icon: Globe, platform: true }],
  },
];

export const ALL_NAV = NAV.flatMap((s) => s.items);

export function titleFor(path: string): string {
  const hit = ALL_NAV.filter((n) => path === n.to || path.startsWith(`${n.to}/`)).sort((a, b) => b.to.length - a.to.length)[0];
  return hit?.label ?? '';
}
