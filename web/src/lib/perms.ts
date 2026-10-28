import type { Role } from '../api/types';

/** Server permissions (domain/permissions.go). */
export const PERMS = {
  orgManage: 'org.manage', usersManage: 'users.manage', queuesManage: 'queues.manage', flowsManage: 'flows.manage',
  aiManage: 'ai.manage', routingManage: 'routing.manage', routingApprove: 'routing.approve', kbManage: 'kb.manage',
  kbRestricted: 'kb.restricted', ticketsAll: 'tickets.all', callsAll: 'calls.all', callsControl: 'calls.control',
  callsMonitor: 'calls.monitor', recordingsPlay: 'recordings.play', recordingsExport: 'recordings.export',
  qaEvaluate: 'qa.evaluate', reportsView: 'reports.view', campaignsManage: 'campaigns.manage',
  settingsManage: 'settings.manage', auditView: 'audit.view',
} as const;
export type Perm = (typeof PERMS)[keyof typeof PERMS];

export const ALL_PERMS: string[] = Object.values(PERMS);

/** Human labels for the role editor. */
export const PERM_LABELS: Record<string, string> = {
  'org.manage': 'Departments, job profiles and revision history',
  'users.manage': 'Employees, invitations and roles',
  'queues.manage': 'Queues, members and schedules',
  'flows.manage': 'IVR flows and phone numbers',
  'ai.manage': 'AI agent profiles',
  'routing.manage': 'Routing topics, draft rules and simulation',
  'routing.approve': 'Approve, reject and retire routing rules',
  'kb.manage': 'Manage the knowledge base',
  'kb.restricted': 'Read restricted knowledge',
  'tickets.all': 'See and manage every ticket',
  'calls.all': 'See every call and the live dashboard',
  'calls.control': 'Hang up, transfer and set status for any call or agent',
  'calls.monitor': 'Listen, whisper and barge',
  'recordings.play': 'Play any recording',
  'recordings.export': 'Download recordings',
  'qa.evaluate': 'Quality scorecards and evaluations',
  'reports.view': 'Reports',
  'campaigns.manage': 'Outbound campaigns',
  'settings.manage': 'Settings, webhooks, email connectors and retention',
  'audit.view': 'Audit log',
};

const SUPERVISOR: string[] = [
  PERMS.ticketsAll, PERMS.callsAll, PERMS.callsControl, PERMS.callsMonitor, PERMS.recordingsPlay,
  PERMS.qaEvaluate, PERMS.reportsView, PERMS.campaignsManage, PERMS.routingManage,
];

/** Built-in role defaults, used when talking to a server that does not report permissions. */
export function defaultPerms(role: Role | undefined): string[] {
  if (role === 'admin') return ALL_PERMS;
  if (role === 'supervisor') return SUPERVISOR;
  return [];
}
