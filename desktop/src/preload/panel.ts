/**
 * Preload for the main window (the remote web panel).
 *
 * Runs sandboxed with context isolation: the page never sees Node or
 * ipcRenderer, only the frozen `window.ccDesktop` object below. The bridge is
 * exposed only when the main process confirms the document belongs to the
 * configured server origin, and every message is re-validated there.
 *
 * Must stay self-contained (no local imports): sandboxed preloads can only
 * require a small set of Electron modules.
 */
import { contextBridge, ipcRenderer } from 'electron';

interface Hello {
  allowed: boolean;
  version: string;
}

let hello: Hello = { allowed: false, version: '' };
try {
  hello = (ipcRenderer.sendSync('cc:hello') as Hello) || hello;
} catch {
  /* main process unavailable: no bridge */
}

if (hello.allowed) {
  const bridge = {
    /** App version, e.g. "0.1.0". Presence of window.ccDesktop means "running in the desktop app". */
    version: String(hello.version),
    platform: process.platform,
    /** Shows a native notification; clicking it focuses the app window. */
    notify(title: unknown, body?: unknown): void {
      ipcRenderer.send('cc:notify', String(title ?? ''), body == null ? '' : String(body));
    },
    /** Sets the dock/taskbar badge (0 clears it). */
    setBadge(count: unknown): void {
      const n = Math.max(0, Math.min(9999, Math.floor(Number(count) || 0)));
      ipcRenderer.send('cc:badge', n);
    },
    /** Shows, restores and focuses the app window. */
    focus(): void {
      ipcRenderer.send('cc:focus');
    },
  };
  contextBridge.exposeInMainWorld('ccDesktop', Object.freeze(bridge));
}
