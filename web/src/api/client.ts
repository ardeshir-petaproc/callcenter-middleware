// Minimal fetch wrapper: bearer auth, JSON, error normalisation and global 401 handling.
const TOKEN_KEY = 'cc.token';
export const API_BASE = '/api/v1';

let token: string | null = readToken();
const unauthorizedListeners = new Set<() => void>();

function readToken(): string | null {
  try { return localStorage.getItem(TOKEN_KEY); } catch { return null; }
}

export function getToken(): string | null { return token; }
export function setToken(t: string | null): void {
  token = t;
  try {
    if (t) localStorage.setItem(TOKEN_KEY, t);
    else localStorage.removeItem(TOKEN_KEY);
  } catch { /* storage unavailable */ }
}
export function onUnauthorized(fn: () => void): () => void {
  unauthorizedListeners.add(fn);
  return () => unauthorizedListeners.delete(fn);
}

export class ApiError extends Error {
  constructor(public status: number, message: string) { super(message); }
}

type Query = Record<string, string | number | boolean | undefined | null>;

export function buildQuery(q?: Query): string {
  if (!q) return '';
  const p = new URLSearchParams();
  for (const [k, v] of Object.entries(q)) {
    if (v === undefined || v === null || v === '') continue;
    p.set(k, String(v));
  }
  const s = p.toString();
  return s ? `?${s}` : '';
}

/** URL with the token as query param, for <audio src>, downloads and WebSocket. */
export function authUrl(path: string, q?: Query): string {
  return API_BASE + path + buildQuery({ ...q, token: token ?? '' });
}

interface Opts { method?: string; body?: unknown; query?: Query; signal?: AbortSignal; noAuthRedirect?: boolean }

export async function api<T>(path: string, opts: Opts = {}): Promise<T> {
  const headers: Record<string, string> = { Accept: 'application/json' };
  if (opts.body !== undefined) headers['Content-Type'] = 'application/json';
  if (token) headers.Authorization = `Bearer ${token}`;
  let res: Response;
  try {
    res = await fetch(API_BASE + path + buildQuery(opts.query), {
      method: opts.method ?? 'GET',
      headers,
      body: opts.body !== undefined ? JSON.stringify(opts.body) : undefined,
      signal: opts.signal,
    });
  } catch (e) {
    if ((e as Error).name === 'AbortError') throw e;
    throw new ApiError(0, 'Network error — is the server reachable?');
  }
  if (res.status === 401 && !opts.noAuthRedirect) {
    setToken(null);
    unauthorizedListeners.forEach((fn) => fn());
  }
  const text = await res.text();
  let data: unknown = null;
  if (text) {
    try { data = JSON.parse(text); } catch { data = text; }
  }
  if (!res.ok) {
    let msg = `${res.status} ${res.statusText}`;
    if (data && typeof data === 'object' && 'error' in data) msg = String((data as { error: unknown }).error);
    else if (typeof data === 'string' && data.trim() && data.length < 200) msg = data.trim();
    throw new ApiError(res.status, msg);
  }
  return data as T;
}

export const get = <T>(path: string, query?: Query, signal?: AbortSignal) => api<T>(path, { query, signal });
export const post = <T>(path: string, body?: unknown) => api<T>(path, { method: 'POST', body: body ?? {} });
export const put = <T>(path: string, body: unknown) => api<T>(path, { method: 'PUT', body });
export const patch = <T>(path: string, body: unknown) => api<T>(path, { method: 'PATCH', body });
export const del = <T = unknown>(path: string) => api<T>(path, { method: 'DELETE' });

export function errMsg(e: unknown): string {
  if (e instanceof Error) return e.message;
  return String(e);
}
