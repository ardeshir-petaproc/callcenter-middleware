/**
 * Call Center desktop client — Electron main process.
 *
 *  - First run / "Change server": local setup window (static/setup.html).
 *  - Main window: the server's web panel, sandboxed, in its own persistent
 *    session partition, with a minimal `window.ccDesktop` bridge.
 *  - Tray: show, status quick-switch, toggles, change server, quit.
 *  - Incoming call notifications from the main process' own WebSocket.
 */
import {
  app,
  BrowserWindow,
  dialog,
  ipcMain,
  Menu,
  nativeImage,
  nativeTheme,
  Notification,
  screen,
  session,
  shell,
  systemPreferences,
  Tray,
  type IpcMainEvent,
  type IpcMainInvokeEvent,
  type NativeImage,
  type Rectangle,
  type Session,
} from 'electron';
import path from 'node:path';
import { getAutostart, HIDDEN_ARG, setAutostart, startedHidden } from './autostart';
import { appIconPng, badgePng, STATUS_COLORS } from './icon';
import { buildAppMenu, buildTrayMenu, trayTooltip, type MenuContext } from './menus';
import {
  checkHealth,
  insecureOriginNeedingFlag,
  isServerUrl,
  normalizeServerUrl,
  originOf,
  panelUrl,
  type CallView,
  type SettableStatus,
} from './server';
import { AgentSession } from './session';
import { SettingsStore } from './settings';

const DEBUG = process.env.CC_DEBUG === '1';
const PANEL_PARTITION = 'persist:panel';
const TOKEN_KEY = 'cc_token';
const TOKEN_POLL_MS = 3000;
const STATIC_DIR = path.join(__dirname, '..', 'static');
const DEFAULT_SIZE = { width: 1280, height: 820 };

function log(...args: unknown[]): void {
  console.log('[cc-desktop]', ...args);
}

function argValue(name: string): string | undefined {
  const prefix = `${name}=`;
  const hit = process.argv.find((a) => a.startsWith(prefix));
  return hit ? hit.slice(prefix.length) : undefined;
}

// ---------------------------------------------------------------------------
// Pre-ready setup
// ---------------------------------------------------------------------------

// Every renderer is sandboxed, whatever its webPreferences say. (Skipped when
// Chromium's OS sandbox is disabled with --no-sandbox, e.g. CI running as
// root, where it would crash the utility processes; the per-window
// `sandbox: true` still applies.)
if (!app.commandLine.hasSwitch('no-sandbox')) app.enableSandbox();
if (process.platform === 'win32') app.setAppUserModelId('com.callcenter.desktop');

const store = new SettingsStore(app.getPath('userData'));

// CC_SERVER_URL / --server=<url> pre-configure the server on first run only.
const presetServer = process.env.CC_SERVER_URL || argValue('--server');
if (!store.get().serverUrl && presetServer) {
  try {
    store.update({ serverUrl: normalizeServerUrl(presetServer) });
  } catch (err) {
    log('ignoring invalid preset server URL:', (err as Error).message);
  }
}

// The softphone needs getUserMedia, which needs a secure context. For a plain
// http:// LAN server, let Chromium treat exactly that one origin as secure.
// Command-line switches must be set before 'ready', hence the relaunch when
// the server is changed to another such origin at runtime.
const insecureOriginFlag = insecureOriginNeedingFlag(store.get().serverUrl);
if (insecureOriginFlag) {
  app.commandLine.appendSwitch('unsafely-treat-insecure-origin-as-secure', insecureOriginFlag);
}

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------

let mainWindow: BrowserWindow | null = null;
let setupWindow: BrowserWindow | null = null;
let tray: Tray | null = null;
let quitting = false;
let trayHintShown = false;
let badgeCount = 0;
let tokenTimer: NodeJS.Timeout | null = null;
let pollInFlight = false;
const liveNotifications = new Set<Notification>();
const notifyTimes: number[] = [];

const serverUrl = (): string | null => store.get().serverUrl;
const serverOrigin = (): string | null => (serverUrl() ? originOf(serverUrl()!) : null);

const icons = {
  app: null as NativeImage | null,
  badge: null as NativeImage | null,
  tray: new Map<string, NativeImage>(),
};

function appIcon(): NativeImage {
  if (!icons.app) icons.app = nativeImage.createFromBuffer(appIconPng(256));
  return icons.app;
}

function badgeIcon(): NativeImage {
  if (!icons.badge) icons.badge = nativeImage.createFromBuffer(badgePng(16));
  return icons.badge;
}

/** Tray icon with a status-colored dot (16px @1x + 32px @2x). */
function trayIcon(status: string | null): NativeImage {
  const key = status ?? 'none';
  let img = icons.tray.get(key);
  if (!img) {
    const dot = status ? (STATUS_COLORS[status] ?? null) : null;
    if (process.platform === 'linux') {
      img = nativeImage.createFromBuffer(appIconPng(32, { dot }));
    } else {
      img = nativeImage.createFromBuffer(appIconPng(16, { dot }), { scaleFactor: 1 });
      img.addRepresentation({ scaleFactor: 2, buffer: appIconPng(32, { dot }) });
    }
    icons.tray.set(key, img);
  }
  return img;
}

const agent = new AgentSession({
  onChange: () => refreshChrome(),
  onScreenpop: (call, queueName) => notifyIncomingCall(call, queueName),
  log,
});

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function openExternalSafe(url: string): void {
  let protocol = '';
  try {
    protocol = new URL(url).protocol;
  } catch {
    return;
  }
  if (protocol === 'http:' || protocol === 'https:' || protocol === 'mailto:') {
    void shell.openExternal(url).catch((err: Error) => log('openExternal failed:', err.message));
  } else {
    log('blocked external URL with protocol', protocol);
  }
}

function backgroundColor(): string {
  return nativeTheme.shouldUseDarkColors ? '#0f172a' : '#f8fafc';
}

function clip(s: unknown, max: number): string {
  const str = typeof s === 'string' ? s : '';
  return str.length > max ? `${str.slice(0, max - 1)}…` : str;
}

function showNotification(title: string, body: string, urgent = false): void {
  if (!Notification.isSupported()) return;
  const n = new Notification({
    title,
    body,
    icon: appIcon(),
    silent: false,
    urgency: urgent ? 'critical' : 'normal',
  });
  // Keep a reference until it closes, otherwise the click handler can be GC'd.
  liveNotifications.add(n);
  n.on('click', () => {
    liveNotifications.delete(n);
    showMain();
  });
  n.on('close', () => liveNotifications.delete(n));
  if (liveNotifications.size > 20) liveNotifications.delete(liveNotifications.values().next().value!);
  n.show();
}

function requestAttention(): void {
  const win = mainWindow;
  if (win && !win.isDestroyed() && !win.isFocused()) win.flashFrame(true);
  if (process.platform === 'darwin') app.dock?.bounce('critical');
}

function notifyIncomingCall(call: CallView, queueName: string | null): void {
  if (!store.get().notifyIncomingCalls) return;
  const win = mainWindow;
  // When the panel is in front of the agent it shows its own screen pop.
  if (win && !win.isDestroyed() && win.isVisible() && win.isFocused()) return;
  const name = clip(call.caller_name, 80);
  const number = clip(call.caller_number, 40);
  const title = `Incoming call${name || number ? `: ${name || number}` : ''}`;
  const lines: string[] = [];
  if (name && number) lines.push(number);
  if (queueName) lines.push(`Queue: ${clip(queueName, 80)}`);
  else if (call.queue_id) lines.push(`Queue #${call.queue_id}`);
  if (!lines.length) lines.push('Click to open the panel');
  log(`incoming call notification: ${title}${queueName ? ` [${queueName}]` : ''}`);
  showNotification(title, lines.join('\n'), true);
  requestAttention();
}

function setBadge(count: number): void {
  badgeCount = count;
  app.setBadgeCount(count); // macOS dock, Linux Unity launcher; no-op elsewhere
  if (process.platform === 'win32' && mainWindow && !mainWindow.isDestroyed()) {
    mainWindow.setOverlayIcon(count > 0 ? badgeIcon() : null, count > 0 ? `${count} pending` : '');
  }
  refreshChrome();
}

// ---------------------------------------------------------------------------
// Menus & tray
// ---------------------------------------------------------------------------

const menuCtx: MenuContext = {
  debug: DEBUG,
  appName: app.getName(),
  serverUrl,
  settings: () => store.get(),
  agent,
  autostart: () => {
    try {
      return getAutostart();
    } catch {
      return false;
    }
  },
  badgeCount: () => badgeCount,
  showMain: () => showMain(),
  openSetup: () => openSetup(),
  reload: (force) => reloadPanel(force),
  quit: () => {
    quitting = true;
    app.quit();
  },
  setStatus: (status, reason) => void changeStatus(status, reason),
  setCloseToTray: (on) => {
    store.update({ closeToTray: on });
    refreshChrome();
  },
  setNotify: (on) => {
    store.update({ notifyIncomingCalls: on });
    refreshChrome();
  },
  setAutostart: (on) => {
    try {
      setAutostart(on);
    } catch (err) {
      void dialog.showMessageBox({
        type: 'error',
        message: 'Could not change the launch-at-login setting',
        detail: (err as Error).message,
      });
    }
    refreshChrome();
  },
  openInBrowser: () => {
    const url = serverUrl();
    if (url) openExternalSafe(panelUrl(url));
  },
  showAbout: () => showAbout(),
};

function refreshChrome(): void {
  Menu.setApplicationMenu(buildAppMenu(menuCtx));
  if (tray && !tray.isDestroyed()) {
    tray.setImage(trayIcon(agent.loggedIn ? (agent.status ?? 'offline') : null));
    tray.setToolTip(trayTooltip(menuCtx));
    tray.setContextMenu(buildTrayMenu(menuCtx));
  }
}

function createTray(): void {
  try {
    tray = new Tray(trayIcon(null));
  } catch (err) {
    log('tray unavailable:', (err as Error).message);
    tray = null;
    return;
  }
  tray.on('click', () => {
    const win = mainWindow;
    if (win && !win.isDestroyed() && win.isVisible() && win.isFocused()) win.hide();
    else showMain();
  });
  refreshChrome();
}

async function changeStatus(status: SettableStatus, reason?: string): Promise<void> {
  try {
    await agent.setStatus(status, reason);
    log(`status changed to ${status}${reason ? ` (${reason})` : ''}`);
  } catch (err) {
    refreshChrome(); // undo the checkbox toggle
    void dialog.showMessageBox({
      type: 'error',
      message: 'Could not change your status',
      detail: (err as Error).message,
    });
  }
}

function showAbout(): void {
  const user = agent.user ? `${agent.user.display_name || agent.user.username} (${agent.user.role})` : 'not signed in';
  void dialog.showMessageBox({
    type: 'info',
    title: `About ${app.getName()}`,
    message: `${app.getName()} ${app.getVersion()}`,
    detail: [
      `Server: ${serverUrl() ?? 'not configured'}`,
      `User: ${user}`,
      `Realtime: ${agent.connected ? 'connected' : 'disconnected'}`,
      '',
      `Electron ${process.versions.electron} · Chromium ${process.versions.chrome} · Node ${process.versions.node}`,
      `Settings: ${store.file}`,
    ].join('\n'),
    icon: appIcon(),
  });
}

// ---------------------------------------------------------------------------
// Panel session: permissions
// ---------------------------------------------------------------------------

function configurePanelSession(ses: Session): void {
  ses.setPermissionRequestHandler((wc, permission, callback, details) => {
    const origin = originOf(details.requestingUrl || wc.getURL());
    const fromServer = origin !== null && origin === serverOrigin();
    let allow = false;
    if (fromServer && permission === 'media') {
      const types = 'mediaTypes' in details && Array.isArray(details.mediaTypes) ? details.mediaTypes : [];
      allow = types.length > 0 && types.every((t) => t === 'audio'); // microphone only, no camera
    } else if (fromServer && permission === 'clipboard-sanitized-write') {
      allow = true; // "copy" buttons in the panel
    }
    if (!allow) {
      log(`denied permission "${permission}" for ${origin ?? 'unknown origin'}`);
      callback(false);
      return;
    }
    if (permission === 'media' && process.platform === 'darwin') {
      systemPreferences
        .askForMediaAccess('microphone')
        .then((granted) => callback(granted))
        .catch(() => callback(false));
      return;
    }
    callback(true);
  });

  ses.setPermissionCheckHandler((_wc, permission, requestingOrigin, details) => {
    if (originOf(requestingOrigin) !== serverOrigin()) return false;
    if (permission === 'media') {
      const t = (details as { mediaType?: string }).mediaType;
      return t === 'audio' || t === undefined || t === 'unknown';
    }
    return permission === 'clipboard-sanitized-write';
  });

  // WebHID / WebSerial / WebUSB: never.
  ses.setDevicePermissionHandler(() => false);
}

// ---------------------------------------------------------------------------
// Main window
// ---------------------------------------------------------------------------

function initialBounds(): Partial<Rectangle> & { width: number; height: number } {
  const w = store.get().window;
  if (!w) return { ...DEFAULT_SIZE };
  const bounds = { width: w.width, height: w.height } as Partial<Rectangle> & { width: number; height: number };
  if (w.x !== undefined && w.y !== undefined) {
    const rect = { x: w.x, y: w.y, width: w.width, height: w.height };
    const area = screen.getDisplayMatching(rect).workArea;
    const overlapX = Math.min(rect.x + rect.width, area.x + area.width) - Math.max(rect.x, area.x);
    const overlapY = Math.min(rect.y + rect.height, area.y + area.height) - Math.max(rect.y, area.y);
    if (overlapX >= 120 && overlapY >= 80) {
      bounds.x = w.x;
      bounds.y = w.y;
    }
    bounds.width = Math.min(bounds.width, area.width);
    bounds.height = Math.min(bounds.height, area.height);
  }
  return bounds;
}

function persistBounds(win: BrowserWindow): void {
  if (win.isDestroyed() || win.isFullScreen()) return;
  const b = win.getNormalBounds();
  store.update({ window: { x: b.x, y: b.y, width: b.width, height: b.height, maximized: win.isMaximized() } });
}

function createMainWindow(show: boolean): BrowserWindow {
  const url = serverUrl();
  if (!url) throw new Error('createMainWindow without a server');
  const ses = session.fromPartition(PANEL_PARTITION);
  const win = new BrowserWindow({
    ...initialBounds(),
    minWidth: 800,
    minHeight: 560,
    show: false,
    title: app.getName(),
    icon: appIcon(),
    backgroundColor: backgroundColor(),
    autoHideMenuBar: true,
    webPreferences: {
      preload: path.join(__dirname, 'preload', 'panel.js'),
      session: ses,
      contextIsolation: true,
      nodeIntegration: false,
      nodeIntegrationInSubFrames: false,
      sandbox: true,
      webSecurity: true,
      webviewTag: false,
      navigateOnDragDrop: false,
      safeDialogs: true,
      spellcheck: true,
      devTools: DEBUG,
      // Keep the softphone, ringing and WebSocket alive while hidden in the tray.
      backgroundThrottling: false,
    },
  });
  mainWindow = win;
  if (store.get().window?.maximized) win.maximize();

  win.once('ready-to-show', () => {
    if (show) {
      win.show();
      win.focus();
    }
  });

  let saveTimer: NodeJS.Timeout | null = null;
  const scheduleSave = () => {
    if (saveTimer) clearTimeout(saveTimer);
    saveTimer = setTimeout(() => persistBounds(win), 500);
  };
  for (const ev of ['resize', 'move', 'maximize', 'unmaximize'] as const) win.on(ev as 'resize', scheduleSave);

  win.on('focus', () => win.flashFrame(false));
  win.on('close', (e) => {
    persistBounds(win);
    if (!quitting && store.get().closeToTray && tray) {
      e.preventDefault();
      win.hide();
      if (!trayHintShown && process.platform !== 'darwin') {
        trayHintShown = true;
        showNotification(app.getName(), 'Still running in the tray. Incoming calls will notify you.');
      }
    }
  });
  win.on('closed', () => {
    if (saveTimer) clearTimeout(saveTimer);
    if (mainWindow === win) mainWindow = null;
  });

  const wc = win.webContents;
  wc.on('did-fail-load', (_e, code, desc, validatedUrl, isMainFrame) => {
    // -3 = ERR_ABORTED (navigation superseded), not a real failure.
    if (!isMainFrame || code === -3 || !isServerUrl(validatedUrl, serverUrl())) return;
    log(`panel failed to load (${code} ${desc})`);
    void win.loadFile(path.join(STATIC_DIR, 'offline.html'), {
      query: { url: validatedUrl, error: `${desc} (${code})` },
    });
    if (!win.isVisible() && show) win.show();
  });
  wc.on('did-finish-load', () => void pollToken());
  wc.on('render-process-gone', (_e, details) => {
    log('panel renderer gone:', details.reason);
    if (details.reason !== 'clean-exit' && !win.isDestroyed()) setTimeout(() => loadPanel(), 1000);
  });

  loadPanel();
  return win;
}

function loadPanel(): void {
  const url = serverUrl();
  const win = mainWindow;
  if (!url || !win || win.isDestroyed()) return;
  log('loading panel', panelUrl(url));
  win.loadURL(panelUrl(url)).catch(() => {
    /* reported through did-fail-load */
  });
}

function reloadPanel(force: boolean): void {
  const win = mainWindow;
  if (!win || win.isDestroyed()) return;
  if (!isServerUrl(win.webContents.getURL(), serverUrl())) loadPanel();
  else if (force) win.webContents.reloadIgnoringCache();
  else win.webContents.reload();
}

function showMain(): void {
  if (!serverUrl()) {
    openSetup();
    return;
  }
  if (!mainWindow || mainWindow.isDestroyed()) {
    createMainWindow(true);
    return;
  }
  if (mainWindow.isMinimized()) mainWindow.restore();
  mainWindow.show();
  mainWindow.focus();
  if (process.platform === 'darwin') app.focus({ steal: true });
}

// ---------------------------------------------------------------------------
// Token polling: the panel owns login; we follow its localStorage token.
// ---------------------------------------------------------------------------

const TOKEN_JS = `(() => { try { return window.localStorage.getItem(${JSON.stringify(TOKEN_KEY)}); } catch (e) { return null; } })()`;

/** The panel's token, null when logged out, undefined when unknown (panel not loaded). */
async function readPanelToken(): Promise<string | null | undefined> {
  const win = mainWindow;
  if (!win || win.isDestroyed()) return null;
  const wc = win.webContents;
  if (wc.isCrashed() || !isServerUrl(wc.getURL(), serverUrl())) return undefined;
  let timer: NodeJS.Timeout | null = null;
  try {
    const value: unknown = await Promise.race([
      wc.executeJavaScript(TOKEN_JS, false),
      new Promise((resolve) => {
        timer = setTimeout(() => resolve(undefined), 3000);
      }),
    ]);
    if (value === undefined) return undefined;
    if (typeof value === 'string' && value.length > 0 && value.length <= 8192 && /^[\x21-\x7e]+$/.test(value)) {
      return value;
    }
    return null;
  } catch {
    return undefined;
  } finally {
    if (timer) clearTimeout(timer);
  }
}

async function pollToken(): Promise<void> {
  if (pollInFlight) return;
  pollInFlight = true;
  try {
    const token = await readPanelToken();
    if (token !== undefined) agent.setToken(token);
  } finally {
    pollInFlight = false;
  }
}

// ---------------------------------------------------------------------------
// Setup window
// ---------------------------------------------------------------------------

function openSetup(): void {
  if (setupWindow && !setupWindow.isDestroyed()) {
    setupWindow.show();
    setupWindow.focus();
    return;
  }
  const parent = mainWindow && !mainWindow.isDestroyed() && mainWindow.isVisible() ? mainWindow : undefined;
  const win = new BrowserWindow({
    width: 540,
    height: 620,
    resizable: false,
    maximizable: false,
    minimizable: !parent,
    fullscreenable: false,
    parent,
    modal: false,
    show: false,
    title: 'Connect to server',
    icon: appIcon(),
    backgroundColor: backgroundColor(),
    autoHideMenuBar: true,
    webPreferences: {
      preload: path.join(__dirname, 'preload', 'setup.js'),
      contextIsolation: true,
      nodeIntegration: false,
      sandbox: true,
      webviewTag: false,
      navigateOnDragDrop: false,
      spellcheck: false,
      devTools: DEBUG,
    },
  });
  setupWindow = win;
  if (process.platform !== 'darwin') win.setMenu(null);
  win.once('ready-to-show', () => win.show());
  win.on('closed', () => {
    if (setupWindow === win) setupWindow = null;
    if (!serverUrl() && !quitting) {
      quitting = true;
      app.quit();
    }
  });
  void win.loadFile(path.join(STATIC_DIR, 'setup.html'));
}

function applyServer(url: string): void {
  const previous = serverUrl();
  store.update({ serverUrl: url });
  log(`server set to ${url}${previous && previous !== url ? ` (was ${previous})` : ''}`);
  if (setupWindow && !setupWindow.isDestroyed()) setupWindow.close();

  const needed = insecureOriginNeedingFlag(url);
  if (needed && needed !== insecureOriginFlag) {
    log('relaunching so the softphone can use the microphone on', needed);
    quitting = true;
    app.relaunch({ args: process.argv.slice(1).filter((a) => a !== HIDDEN_ARG && !a.startsWith('--server=')) });
    app.exit(0);
    return;
  }
  agent.setServer(url);
  if (mainWindow && !mainWindow.isDestroyed()) {
    loadPanel();
    showMain();
  } else {
    createMainWindow(true);
  }
  refreshChrome();
}

// ---------------------------------------------------------------------------
// IPC
// ---------------------------------------------------------------------------

/** True when the message comes from the top frame of the main window on the server origin. */
function fromPanel(e: IpcMainEvent): boolean {
  const win = mainWindow;
  const frame = e.senderFrame;
  return (
    !!win &&
    !win.isDestroyed() &&
    e.sender === win.webContents &&
    !!frame &&
    frame.parent === null &&
    isServerUrl(frame.url, serverUrl())
  );
}

function fromSetup(e: IpcMainEvent | IpcMainInvokeEvent): boolean {
  return !!setupWindow && !setupWindow.isDestroyed() && e.sender === setupWindow.webContents;
}

function registerIpc(): void {
  // --- panel bridge (window.ccDesktop) ---
  ipcMain.on('cc:hello', (e) => {
    e.returnValue = { allowed: fromPanel(e), version: app.getVersion() };
  });
  ipcMain.on('cc:notify', (e, title: unknown, body: unknown) => {
    if (!fromPanel(e)) return;
    const now = Date.now();
    while (notifyTimes.length && now - notifyTimes[0] > 10000) notifyTimes.shift();
    if (notifyTimes.length >= 5) return; // max 5 notifications per 10 s
    notifyTimes.push(now);
    showNotification(clip(title, 120) || app.getName(), clip(body, 500));
  });
  ipcMain.on('cc:badge', (e, count: unknown) => {
    if (!fromPanel(e)) return;
    const n = Math.max(0, Math.min(9999, Math.floor(Number(count) || 0)));
    if (n !== badgeCount) setBadge(n);
  });
  ipcMain.on('cc:focus', (e) => {
    if (fromPanel(e)) showMain();
  });

  // --- setup window ---
  ipcMain.handle('setup:info', (e) => {
    if (!fromSetup(e)) throw new Error('forbidden');
    return { serverUrl: serverUrl(), firstRun: !serverUrl(), version: app.getVersion() };
  });
  ipcMain.handle('setup:check', async (e, raw: unknown) => {
    if (!fromSetup(e)) throw new Error('forbidden');
    let url: string;
    try {
      url = normalizeServerUrl(String(raw ?? ''));
    } catch (err) {
      return { ok: false, error: (err as Error).message };
    }
    return { ...(await checkHealth(url)), url };
  });
  ipcMain.handle('setup:save', async (e, raw: unknown, force: unknown) => {
    if (!fromSetup(e)) throw new Error('forbidden');
    let url: string;
    try {
      url = normalizeServerUrl(String(raw ?? ''));
    } catch (err) {
      return { ok: false, error: (err as Error).message };
    }
    if (force !== true) {
      const health = await checkHealth(url);
      if (!health.ok) return { ok: false, url, error: health.error, canForce: true };
    }
    setImmediate(() => applyServer(url));
    return { ok: true, url };
  });
  ipcMain.on('setup:cancel', (e) => {
    if (fromSetup(e)) setupWindow?.close();
  });
}

// ---------------------------------------------------------------------------
// Global hardening for every webContents
// ---------------------------------------------------------------------------

app.on('web-contents-created', (_e, contents) => {
  if (DEBUG) contents.on('console-message', (ev) => log(`[renderer ${ev.level}] ${ev.message}`));
  contents.on('will-attach-webview', (ev) => ev.preventDefault());
  // target=_blank / window.open: never a new Electron window; http(s)/mailto go to the browser.
  contents.setWindowOpenHandler(({ url }) => {
    openExternalSafe(url);
    return { action: 'deny' };
  });
  contents.on('will-navigate', (ev, url) => {
    const isMain = !!mainWindow && !mainWindow.isDestroyed() && contents === mainWindow.webContents;
    if (isMain && isServerUrl(url, serverUrl())) return;
    ev.preventDefault();
    openExternalSafe(url);
  });
});

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

function init(): void {
  log(`starting ${app.getName()} ${app.getVersion()} (electron ${process.versions.electron})${DEBUG ? ' [debug]' : ''}`);
  if (insecureOriginFlag) log('treating', insecureOriginFlag, 'as a secure origin (softphone microphone access)');
  registerIpc();
  configurePanelSession(session.fromPartition(PANEL_PARTITION));
  createTray();
  refreshChrome();

  const url = serverUrl();
  if (!url) {
    openSetup();
  } else {
    agent.setServer(url);
    const hidden = startedHidden() && !!tray;
    createMainWindow(!hidden);
  }
  tokenTimer = setInterval(() => void pollToken(), TOKEN_POLL_MS);
  nativeTheme.on('updated', () => {
    for (const w of BrowserWindow.getAllWindows()) w.setBackgroundColor(backgroundColor());
  });
}

if (!app.requestSingleInstanceLock()) {
  app.quit();
} else {
  app.on('second-instance', () => showMain());
  app.on('activate', () => showMain()); // macOS dock click
  app.on('before-quit', () => {
    quitting = true;
    if (mainWindow && !mainWindow.isDestroyed()) persistBounds(mainWindow);
  });
  app.on('will-quit', () => {
    if (tokenTimer) clearInterval(tokenTimer);
    agent.stop();
  });
  app.on('window-all-closed', () => {
    // Close-to-tray hides instead of closing, so getting here means a real close.
    if (process.platform !== 'darwin') app.quit();
  });
  app
    .whenReady()
    .then(init)
    .catch((err: Error) => {
      console.error('[cc-desktop] fatal startup error:', err);
      app.exit(1);
    });
}
