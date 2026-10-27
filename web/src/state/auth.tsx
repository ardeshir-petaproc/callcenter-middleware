import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react';
import { api, get, getToken, onUnauthorized, setToken } from '../api/client';
import type { AgentState, LoginResponse, MeResponse, User } from '../api/types';
import { applyAgent, live, resetLive } from './live';
import { startLive, stopLive } from './ws';
import { clearLookups } from './lookups';

interface AuthCtx {
  user: User | null;
  ready: boolean;
  login: (username: string, password: string) => Promise<User>;
  logout: () => void;
  refresh: () => Promise<void>;
}

const Ctx = createContext<AuthCtx | null>(null);
const USER_KEY = 'cc.user';

function cachedUser(): User | null {
  try {
    const raw = localStorage.getItem(USER_KEY);
    return raw && getToken() ? (JSON.parse(raw) as User) : null;
  } catch { return null; }
}

function storeUser(u: User | null) {
  try {
    if (u) localStorage.setItem(USER_KEY, JSON.stringify(u));
    else localStorage.removeItem(USER_KEY);
  } catch { /* ignore */ }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(cachedUser);
  const [ready, setReady] = useState(!getToken());

  const logout = useCallback(() => {
    setToken(null);
    storeUser(null);
    stopLive();
    clearLookups();
    setUser(null);
    setReady(true);
  }, []);

  const begin = useCallback((u: User, agent?: AgentState | null) => {
    storeUser(u);
    setUser(u);
    if (live.get().meId !== u.id) resetLive(u.id, u.role);
    else live.set({ meRole: u.role });
    if (agent) applyAgent(agent);
    startLive();
  }, []);

  const refresh = useCallback(async () => {
    const me = await get<MeResponse>('/auth/me');
    begin(me.user, me.agent);
  }, [begin]);

  useEffect(() => onUnauthorized(() => logout()), [logout]);

  useEffect(() => {
    if (!getToken()) return;
    refresh()
      .catch(() => { /* 401 → logout via listener; network errors keep cached session */ })
      .finally(() => setReady(true));
    return () => stopLive();
  }, [refresh]);

  const login = useCallback(async (username: string, password: string) => {
    const res = await api<LoginResponse>('/auth/login', { method: 'POST', body: { username, password }, noAuthRedirect: true });
    setToken(res.token);
    begin(res.user);
    void get<MeResponse>('/auth/me').then((me) => me.agent && applyAgent(me.agent)).catch(() => undefined);
    return res.user;
  }, [begin]);

  const value = useMemo(() => ({ user, ready, login, logout, refresh }), [user, ready, login, logout, refresh]);
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
