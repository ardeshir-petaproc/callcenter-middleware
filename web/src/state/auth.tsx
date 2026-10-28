import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import { api, get, getToken, onUnauthorized, post, setToken } from '../api/client';
import type { AgentState, LoginResponse, MeResponse, Tenant, User } from '../api/types';
import type { ChooseTenant } from '../api/types_org';
import { defaultPerms } from '../lib/perms';
import { applyAgent, live, resetLive } from './live';
import { startLive, stopLive } from './ws';
import { clearLookups } from './lookups';

interface Session { user: User; tenant: Tenant | null; permissions: string[] | null }

interface AuthCtx {
  user: User | null;
  tenant: Tenant | null;
  permissions: string[];
  ready: boolean;
  /** True when the signed-in user holds the permission (role defaults on older servers). */
  can: (perm: string) => boolean;
  login: (username: string, password: string, tenant?: string) => Promise<User>;
  /** Signs in with a Joow token; returns the organization list when the account must choose one. */
  loginJoow: (token: string, tenantId?: number) => Promise<ChooseTenant | null>;
  /** Starts a session from a login-shaped response (e.g. accepting an invitation). */
  adopt: (res: LoginResponse) => void;
  switchTenant: (tenantId: number) => Promise<void>;
  logout: () => void;
  refresh: () => Promise<void>;
}

const Ctx = createContext<AuthCtx | null>(null);
const SESSION_KEY = 'cc.session';

function cachedSession(): Session | null {
  try {
    const raw = localStorage.getItem(SESSION_KEY);
    return raw && getToken() ? (JSON.parse(raw) as Session) : null;
  } catch { return null; }
}

function storeSession(s: Session | null) {
  try {
    if (s) localStorage.setItem(SESSION_KEY, JSON.stringify(s));
    else localStorage.removeItem(SESSION_KEY);
    localStorage.removeItem('cc.user'); // legacy key
  } catch { /* ignore */ }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [session, setSession] = useState<Session | null>(cachedSession);
  const [ready, setReady] = useState(!getToken());

  const logout = useCallback(() => {
    setToken(null);
    storeSession(null);
    stopLive();
    clearLookups();
    setSession(null);
    setReady(true);
  }, []);

  const begin = useCallback((s: Session, agent?: AgentState | null) => {
    storeSession(s);
    setSession(s);
    const u = s.user;
    if (live.get().meId !== u.id) resetLive(u.id, u.role);
    else live.set({ meRole: u.role });
    if (agent) applyAgent(agent);
    startLive();
  }, []);

  const refresh = useCallback(async () => {
    const me = await get<MeResponse>('/auth/me');
    begin({ user: me.user, tenant: me.tenant ?? null, permissions: me.permissions ?? null }, me.agent);
  }, [begin]);

  useEffect(() => onUnauthorized(() => logout()), [logout]);

  useEffect(() => {
    if (!getToken()) return;
    refresh()
      .catch(() => { /* 401 → logout via listener; network errors keep cached session */ })
      .finally(() => setReady(true));
    return () => stopLive();
  }, [refresh]);

  const adopt = useCallback((res: LoginResponse) => {
    setToken(res.token);
    begin({ user: res.user, tenant: res.tenant ?? null, permissions: res.permissions ?? null });
    void get<MeResponse>('/auth/me').then((me) => me.agent && applyAgent(me.agent)).catch(() => undefined);
  }, [begin]);

  const login = useCallback(async (username: string, password: string, tenant?: string) => {
    const res = await api<LoginResponse>('/auth/login', { method: 'POST', body: { username, password, tenant: tenant || undefined }, noAuthRedirect: true });
    adopt(res);
    return res.user;
  }, [adopt]);

  const loginJoow = useCallback(async (token: string, tenantId?: number) => {
    const res = await api<LoginResponse | ChooseTenant>('/auth/joow', { method: 'POST', body: { token, tenant_id: tenantId || undefined }, noAuthRedirect: true });
    if ('choose_tenant' in res && res.choose_tenant) return res;
    adopt(res as LoginResponse);
    return null;
  }, [adopt]);

  const switchTenant = useCallback(async (tenantId: number) => {
    const res = await post<LoginResponse>('/auth/switch', { tenant_id: tenantId });
    setToken(res.token);
    storeSession({ user: res.user, tenant: res.tenant ?? null, permissions: res.permissions ?? null });
    stopLive();
    clearLookups();
    // A fresh page load guarantees no data from the previous organization survives in memory.
    window.location.assign('/');
  }, []);

  const value = useMemo<AuthCtx>(() => {
    const permissions = session ? session.permissions ?? defaultPerms(session.user.role) : [];
    const set = new Set(permissions);
    return {
      user: session?.user ?? null, tenant: session?.tenant ?? null, permissions, ready,
      can: (p: string) => set.has(p), login, loginJoow, adopt, switchTenant, logout, refresh,
    };
  }, [session, ready, login, loginJoow, adopt, switchTenant, logout, refresh]);
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useAuth(): AuthCtx {
  const c = useContext(Ctx);
  if (!c) throw new Error('useAuth outside AuthProvider');
  return c;
}

/** The logged-in user; only call inside authenticated routes. */
export function useMe(): User {
  const { user } = useAuth();
  if (!user) throw new Error('not authenticated');
  return user;
}

/** Permission check bound to the current session. */
export function useCan(): (perm: string) => boolean {
  return useAuth().can;
}
