/**
 * Tiny JSON settings store in app.getPath('userData')/settings.json.
 * Writes are atomic (temp file + rename) and unknown/invalid fields are
 * dropped on load so a hand-edited file can never crash the app.
 */
import fs from 'node:fs';
import path from 'node:path';

export interface WindowState {
  x?: number;
  y?: number;
  width: number;
  height: number;
  maximized: boolean;
}

export interface Settings {
  /** Normalized server base URL (no trailing slash), or null before first run. */
  serverUrl: string | null;
  /** Closing the main window hides it to the tray instead of quitting. */
  closeToTray: boolean;
  /** Show native notifications for incoming calls (agent.screenpop). */
  notifyIncomingCalls: boolean;
  /** Last main window geometry. */
  window: WindowState | null;
}

export const DEFAULT_SETTINGS: Settings = {
  serverUrl: null,
  closeToTray: true,
  notifyIncomingCalls: true,
  window: null,
};

const isNum = (v: unknown): v is number => typeof v === 'number' && Number.isFinite(v);

function sanitize(raw: unknown): Settings {
  const s: Settings = { ...DEFAULT_SETTINGS };
  if (!raw || typeof raw !== 'object') return s;
  const r = raw as Record<string, unknown>;
  if (typeof r.serverUrl === 'string' && r.serverUrl.length > 0) s.serverUrl = r.serverUrl;
  if (typeof r.closeToTray === 'boolean') s.closeToTray = r.closeToTray;
  if (typeof r.notifyIncomingCalls === 'boolean') s.notifyIncomingCalls = r.notifyIncomingCalls;
  const w = r.window as Record<string, unknown> | null | undefined;
  if (w && typeof w === 'object' && isNum(w.width) && isNum(w.height)) {
    s.window = {
      width: Math.max(400, Math.round(w.width)),
      height: Math.max(300, Math.round(w.height)),
      maximized: w.maximized === true,
    };
    if (isNum(w.x) && isNum(w.y)) {
      s.window.x = Math.round(w.x);
      s.window.y = Math.round(w.y);
    }
  }
  return s;
}

export class SettingsStore {
  readonly file: string;
  private data: Settings;

  constructor(dir: string) {
    this.file = path.join(dir, 'settings.json');
    this.data = this.load();
  }

  private load(): Settings {
    try {
      return sanitize(JSON.parse(fs.readFileSync(this.file, 'utf8')));
    } catch (err) {
      if ((err as NodeJS.ErrnoException).code !== 'ENOENT') {
        console.warn('[settings] ignoring unreadable settings file:', (err as Error).message);
      }
      return { ...DEFAULT_SETTINGS };
    }
  }

  get(): Readonly<Settings> {
    return this.data;
  }

  update(patch: Partial<Settings>): Readonly<Settings> {
    this.data = sanitize({ ...this.data, ...patch });
    this.save();
    return this.data;
  }

  private save(): void {
    try {
      fs.mkdirSync(path.dirname(this.file), { recursive: true });
      const tmp = `${this.file}.${process.pid}.tmp`;
      fs.writeFileSync(tmp, JSON.stringify(this.data, null, 2) + '\n', { mode: 0o600 });
      fs.renameSync(tmp, this.file);
    } catch (err) {
      console.error('[settings] failed to save:', (err as Error).message);
    }
  }
}
