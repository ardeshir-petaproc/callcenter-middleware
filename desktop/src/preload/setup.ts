/**
 * Preload for the local connection/setup window (static/setup.html).
 * Sandboxed and context-isolated; exposes only the calls the form needs.
 * Must stay self-contained (no local imports).
 */
import { contextBridge, ipcRenderer } from 'electron';

contextBridge.exposeInMainWorld(
  'ccSetup',
  Object.freeze({
    /** {serverUrl, firstRun, version} */
    info: () => ipcRenderer.invoke('setup:info'),
    /** Validates the URL and probes GET <url>/healthz. */
    check: (url: string) => ipcRenderer.invoke('setup:check', String(url)),
    /** Validates, probes (unless force) and saves the server, then opens the panel. */
    save: (url: string, force: boolean) => ipcRenderer.invoke('setup:save', String(url), force === true),
    /** Closes the dialog (quits when no server is configured yet). */
    cancel: () => ipcRenderer.send('setup:cancel'),
  }),
);
