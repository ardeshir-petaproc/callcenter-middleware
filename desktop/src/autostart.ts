/**
 * Launch-at-login.
 *
 * macOS / Windows: app.setLoginItemSettings (native login items / registry).
 * Linux: Electron's login item API is not implemented there, so we write an
 * XDG autostart entry (~/.config/autostart/<name>.desktop) ourselves.
 *
 * The app is started with `--hidden` so it goes straight to the tray.
 */
import { app } from 'electron';
import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';

export const HIDDEN_ARG = '--hidden';

function linuxAutostartFile(): string {
  const base = process.env.XDG_CONFIG_HOME || path.join(os.homedir(), '.config');
  return path.join(base, 'autostart', 'callcenter-desktop.desktop');
}

/** Quotes one argument for a .desktop Exec= line (freedesktop spec). */
function desktopQuote(arg: string): string {
  return `"${arg.replace(/(["`$\\])/g, '\\$1')}"`;
}

function linuxExecLine(): string {
  // AppImage: the mounted path changes every run; APPIMAGE is the stable file.
  const exe = process.env.APPIMAGE || process.execPath;
  const parts = [exe];
  if (!app.isPackaged) parts.push(app.getAppPath()); // `electron <app dir>` in development
  parts.push(HIDDEN_ARG);
  return parts.map(desktopQuote).join(' ');
}

export function isAutostartSupported(): boolean {
  return process.platform === 'darwin' || process.platform === 'win32' || process.platform === 'linux';
}

export function getAutostart(): boolean {
  if (process.platform === 'linux') return fs.existsSync(linuxAutostartFile());
  try {
    return app.getLoginItemSettings({ args: [HIDDEN_ARG] }).openAtLogin;
  } catch {
    return false;
  }
}

export function setAutostart(enabled: boolean): void {
  if (process.platform === 'linux') {
    const file = linuxAutostartFile();
    if (!enabled) {
      fs.rmSync(file, { force: true });
      return;
    }
    fs.mkdirSync(path.dirname(file), { recursive: true });
    const entry = [
      '[Desktop Entry]',
      'Type=Application',
      `Name=${app.getName()}`,
      'Comment=Call center desktop client',
      `Exec=${linuxExecLine()}`,
      'Terminal=false',
      'X-GNOME-Autostart-enabled=true',
      '',
    ].join('\n');
    fs.writeFileSync(file, entry, { mode: 0o644 });
    return;
  }
  app.setLoginItemSettings({ openAtLogin: enabled, args: [HIDDEN_ARG] });
}

/** True when this process was started by the login item / autostart entry. */
export function startedHidden(): boolean {
  if (process.argv.includes(HIDDEN_ARG)) return true;
  if (process.platform === 'darwin') {
    try {
      return app.getLoginItemSettings().wasOpenedAtLogin === true;
    } catch {
      return false;
    }
  }
  return false;
}
