/** Application menu and tray context menu templates. */
import { Menu, type MenuItemConstructorOptions } from 'electron';
import type { AgentSession } from './session';
import type { AgentStatus, SettableStatus } from './server';
import type { Settings } from './settings';

export interface MenuContext {
  debug: boolean;
  appName: string;
  serverUrl(): string | null;
  settings(): Readonly<Settings>;
  agent: AgentSession;
  autostart(): boolean;
  badgeCount(): number;
  showMain(): void;
  openSetup(): void;
  reload(force: boolean): void;
  quit(): void;
  setStatus(status: SettableStatus, reason?: string): void;
  setCloseToTray(on: boolean): void;
  setNotify(on: boolean): void;
  setAutostart(on: boolean): void;
  openInBrowser(): void;
  showAbout(): void;
}

const STATUS_LABELS: Record<AgentStatus, string> = {
  available: 'Available',
  ringing: 'Ringing',
  busy: 'On a call',
  wrapup: 'Wrap-up',
  paused: 'Paused',
  offline: 'Offline',
};

export function statusLabel(status: AgentStatus | null): string {
  return status ? (STATUS_LABELS[status] ?? status) : 'Unknown';
}

/** One-line description of the signed-in user, used by the tray. */
export function whoLine(ctx: MenuContext): string {
  const a = ctx.agent;
  if (!ctx.serverUrl()) return 'No server configured';
  if (!a.user) return 'Not signed in';
  const name = a.user.display_name || a.user.username;
  const reason = a.status === 'paused' && a.pauseReason ? ` (${a.pauseReason})` : '';
  const link = a.connected ? '' : ' · reconnecting…';
  return `${name} — ${statusLabel(a.status)}${reason}${link}`;
}

export function trayTooltip(ctx: MenuContext): string {
  const parts = [ctx.appName, whoLine(ctx)];
  const n = ctx.badgeCount();
  if (n > 0) parts.push(`${n} pending`);
  return parts.join('\n');
}

export function buildTrayMenu(ctx: MenuContext): Menu {
  const a = ctx.agent;
  const loggedIn = a.loggedIn;
  const s = ctx.settings();
  // Checkbox (not radio) items: a radio group always forces one item checked,
  // which would be wrong while the agent is ringing/busy/in wrap-up.
  const statusItem = (status: SettableStatus): MenuItemConstructorOptions => ({
    label: STATUS_LABELS[status],
    type: 'checkbox',
    checked: a.status === status,
    enabled: loggedIn,
    click: () => ctx.setStatus(status),
  });
  const template: MenuItemConstructorOptions[] = [
    { label: `Show ${ctx.appName}`, click: () => ctx.showMain() },
    { type: 'separator' },
    { label: whoLine(ctx), enabled: false },
    statusItem('available'),
    statusItem('paused'),
  ];
  if (loggedIn && a.pauseReasons.length > 0) {
    template.push({
      label: 'Pause with reason',
      submenu: a.pauseReasons.map((r) => ({
        label: r.label,
        type: 'checkbox' as const,
        checked: a.status === 'paused' && a.pauseReason === r.label,
        click: () => ctx.setStatus('paused', r.label),
      })),
    });
  }
  template.push(
    statusItem('offline'),
    { type: 'separator' },
    {
      label: 'Notify on incoming calls',
      type: 'checkbox',
      checked: s.notifyIncomingCalls,
      click: (item) => ctx.setNotify(item.checked),
    },
    {
      label: 'Close window to tray',
      type: 'checkbox',
      checked: s.closeToTray,
      click: (item) => ctx.setCloseToTray(item.checked),
    },
    {
      label: 'Launch at login',
      type: 'checkbox',
      checked: ctx.autostart(),
      click: (item) => ctx.setAutostart(item.checked),
    },
    { type: 'separator' },
    { label: 'Change server…', click: () => ctx.openSetup() },
    { label: 'Reload panel', enabled: !!ctx.serverUrl(), click: () => ctx.reload(false) },
    { type: 'separator' },
    { label: 'Quit', click: () => ctx.quit() },
  );
  return Menu.buildFromTemplate(template);
}

export function buildAppMenu(ctx: MenuContext): Menu {
  const isMac = process.platform === 'darwin';
  const hasServer = !!ctx.serverUrl();
  const template: MenuItemConstructorOptions[] = [];
  if (isMac) template.push({ role: 'appMenu' });
  template.push(
    {
      label: 'File',
      submenu: [
        { label: 'Change Server…', click: () => ctx.openSetup() },
        { label: 'Open Panel in Browser', enabled: hasServer, click: () => ctx.openInBrowser() },
        { type: 'separator' },
        {
          label: 'Close Window to Tray',
          type: 'checkbox',
          checked: ctx.settings().closeToTray,
          click: (item) => ctx.setCloseToTray(item.checked),
        },
        { type: 'separator' },
        isMac ? { role: 'close' } : { label: 'Quit', accelerator: 'CmdOrCtrl+Q', click: () => ctx.quit() },
      ],
    },
    { role: 'editMenu' },
    {
      label: 'View',
      submenu: [
        { label: 'Reload', accelerator: 'CmdOrCtrl+R', enabled: hasServer, click: () => ctx.reload(false) },
        {
          label: 'Force Reload',
          accelerator: 'CmdOrCtrl+Shift+R',
          enabled: hasServer,
          click: () => ctx.reload(true),
        },
        ...(ctx.debug ? ([{ role: 'toggleDevTools' }] as MenuItemConstructorOptions[]) : []),
        { type: 'separator' },
        { role: 'resetZoom' },
        { role: 'zoomIn' },
        // Ctrl+= is what most keyboards produce for "zoom in".
        { role: 'zoomIn', accelerator: 'CmdOrCtrl+=', visible: false },
        { role: 'zoomOut' },
        { type: 'separator' },
        { role: 'togglefullscreen' },
      ],
    },
    { role: 'windowMenu' },
    {
      role: 'help',
      submenu: [{ label: `About ${ctx.appName}`, click: () => ctx.showAbout() }],
    },
  );
  return Menu.buildFromTemplate(template);
}
