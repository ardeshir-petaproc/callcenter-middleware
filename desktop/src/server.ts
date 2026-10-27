/**
 * Server URL handling and REST helpers for the call center middleware API
 * (see docs/API.md). HTTP goes through Electron's `net.fetch`, i.e. the
 * Chromium network stack, so system proxy settings and the OS certificate
 * store apply exactly as they do for the panel itself.
 */
import { net } from 'electron';

export class ApiError extends Error {
  constructor(
    message: string,
    readonly status: number,
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

/**
 * Normalizes user input into a server base URL: adds http:// when no scheme is
 * given, only allows http/https, rejects credentials, and strips query, hash
 * and trailing slashes. A path prefix is kept (reverse proxy sub-path).
 * Throws an Error with a user-facing message when the input is invalid.
 */
export function normalizeServerUrl(input: string): string {
  let s = String(input ?? '').trim();
  if (!s) throw new Error('Enter the server address, e.g. http://pbx.example.com:8080');
  if (!/^[a-z][a-z0-9+.-]*:\/\//i.test(s)) s = `http://${s}`;
  let u: URL;
  try {
    u = new URL(s);
  } catch {
    throw new Error('That does not look like a valid URL.');
  }
  if (u.protocol !== 'http:' && u.protocol !== 'https:') {
    throw new Error('Only http:// and https:// server addresses are supported.');
  }
  if (u.username || u.password) throw new Error('Do not put credentials in the server address.');
  if (!u.hostname) throw new Error('The server address needs a host name.');
  const pathname = u.pathname.replace(/\/+$/, '');
  return `${u.origin}${pathname}`;
}

export function originOf(url: string): string | null {
  try {
    const o = new URL(url).origin;
    return o === 'null' ? null : o;
  } catch {
    return null;
  }
}

/** True when `url` belongs to the same origin as the configured server. */
export function isServerUrl(url: string, serverUrl: string | null): boolean {
  if (!serverUrl) return false;
  const a = originOf(url);
  return a !== null && a === originOf(serverUrl);
}

/** Hosts that Chromium already treats as potentially trustworthy. */
function isLoopbackHost(hostname: string): boolean {
  const h = hostname.replace(/^\[|\]$/g, '').toLowerCase();
  return h === 'localhost' || h.endsWith('.localhost') || h === '::1' || /^127\.\d+\.\d+\.\d+$/.test(h);
}

/**
 * getUserMedia (the WebRTC softphone) needs a secure context. For a plain
 * http:// server on the LAN we ask Chromium to treat exactly that origin as
 * secure; returns the origin when the flag is required, otherwise null.
 */
export function insecureOriginNeedingFlag(serverUrl: string | null): string | null {
  if (!serverUrl) return null;
  try {
    const u = new URL(serverUrl);
    if (u.protocol === 'http:' && !isLoopbackHost(u.hostname)) return u.origin;
  } catch {
    /* ignore */
  }
  return null;
}

export function panelUrl(serverUrl: string): string {
  return `${serverUrl}/`;
}

export function apiUrl(serverUrl: string, path: string): string {
  return `${serverUrl}/api/v1${path}`;
}

export function wsUrl(serverUrl: string, token: string): string {
  const u = new URL(apiUrl(serverUrl, '/ws'));
  u.protocol = u.protocol === 'https:' ? 'wss:' : 'ws:';
  u.searchParams.set('token', token);
  return u.toString();
}

async function fetchWithTimeout(url: string, init: RequestInit, timeoutMs: number): Promise<Response> {
  const ctrl = new AbortController();
  const timer = setTimeout(() => ctrl.abort(), timeoutMs);
  try {
    return await net.fetch(url, { ...init, signal: ctrl.signal, redirect: 'follow' });
  } catch (err) {
    if (ctrl.signal.aborted) throw new Error(`Timed out after ${Math.round(timeoutMs / 1000)} s`);
    throw err;
  } finally {
    clearTimeout(timer);
  }
}

export type HealthResult = { ok: true } | { ok: false; error: string };

/** GET <server>/healthz — the server is reachable when it answers 2xx. */
export async function checkHealth(serverUrl: string, timeoutMs = 6000): Promise<HealthResult> {
  try {
    const res = await fetchWithTimeout(`${serverUrl}/healthz`, { method: 'GET' }, timeoutMs);
    if (!res.ok) return { ok: false, error: `Server answered HTTP ${res.status} on /healthz` };
    return { ok: true };
  } catch (err) {
    const msg = (err as Error).message || String(err);
    return { ok: false, error: `Could not reach the server: ${msg}` };
  }
}

/** Authenticated JSON request against /api/v1. */
export async function apiRequest<T>(
  serverUrl: string,
  token: string,
  method: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE',
  path: string,
  body?: unknown,
  timeoutMs = 10000,
): Promise<T> {
  const headers: Record<string, string> = { Authorization: `Bearer ${token}`, Accept: 'application/json' };
  const init: RequestInit = { method, headers };
  if (body !== undefined) {
    headers['Content-Type'] = 'application/json';
    init.body = JSON.stringify(body);
  }
  const res = await fetchWithTimeout(apiUrl(serverUrl, path), init, timeoutMs);
  const text = await res.text();
  let json: unknown = null;
  try {
    json = text ? JSON.parse(text) : null;
  } catch {
    /* non-JSON body */
  }
  if (!res.ok) {
    const msg =
      json && typeof json === 'object' && typeof (json as { error?: unknown }).error === 'string'
        ? (json as { error: string }).error
        : `HTTP ${res.status}`;
    throw new ApiError(msg, res.status);
  }
  return json as T;
}

// ---------------------------------------------------------------------------
// Subset of the API entity shapes used by the desktop app (docs/API.md).
// ---------------------------------------------------------------------------

export type AgentStatus = 'offline' | 'available' | 'ringing' | 'busy' | 'wrapup' | 'paused';
export type SettableStatus = 'available' | 'paused' | 'offline';

export interface User {
  id: number;
  username: string;
  display_name: string;
  role: 'admin' | 'supervisor' | 'agent';
  extension?: string;
}

export interface AgentState {
  user_id: number;
  status: AgentStatus;
  pause_reason?: string;
}

export interface MeResponse {
  user: User;
  agent: AgentState | null;
}

export interface PauseReason {
  id: number;
  label: string;
  enabled: boolean;
}

export interface CallView {
  id: string;
  direction?: 'inbound' | 'outbound';
  caller_number?: string;
  caller_name?: string;
  dnis?: string;
  queue_id?: number;
}

export interface QueueView {
  id: number;
  name: string;
}

export interface WsMessage {
  type: string;
  data: unknown;
  at?: string;
}
