import { useEffect, useMemo } from 'react';
import { get } from '../api/client';
import type { AIProfile, Campaign, Disposition, IVRFlow, PauseReason, Queue, Schedule, User } from '../api/types';
import { createStore, useStore } from '../lib/store';
import { onLive } from './ws';
import { live } from './live';

// Cached reference data used for name resolution and pickers across pages.
interface Lookups {
  queues: Queue[];
  users: User[];
  aiProfiles: AIProfile[];
  flows: IVRFlow[];
  schedules: Schedule[];
  dispositions: Disposition[];
  pauseReasons: PauseReason[];
  campaigns: Campaign[];
}
type Key = keyof Lookups;

const paths: Record<Key, string> = {
  queues: '/queues', users: '/users', aiProfiles: '/ai/profiles', flows: '/flows',
  schedules: '/schedules', dispositions: '/dispositions', pauseReasons: '/pause-reasons', campaigns: '/campaigns',
};

const empty: Lookups = { queues: [], users: [], aiProfiles: [], flows: [], schedules: [], dispositions: [], pauseReasons: [], campaigns: [] };
const store = createStore<Lookups>(empty);
const loaded = new Set<Key>();
const inflight = new Map<Key, Promise<void>>();

export function loadLookup(key: Key, force = false): Promise<void> {
  if (!force && loaded.has(key)) return Promise.resolve();
  const existing = inflight.get(key);
  if (existing && !force) return existing;
  const p = get<Lookups[Key]>(paths[key])
    .then((data) => { loaded.add(key); store.set({ [key]: Array.isArray(data) ? data : [] } as Partial<Lookups>); })
    .catch(() => { /* forbidden or unavailable: keep empty */ })
    .finally(() => inflight.delete(key));
  inflight.set(key, p);
  return p;
}

export function refreshLookup(key: Key) { return loadLookup(key, true); }

export function clearLookups() {
  loaded.clear();
  store.set(empty);
}

// Any config change: refetch whatever has been loaded so far.
onLive('config.changed', () => { for (const k of [...loaded]) void loadLookup(k, true); });

export function useLookup<K extends Key>(key: K): Lookups[K] {
  useEffect(() => { void loadLookup(key); }, [key]);
  return useStore(store, (s) => s[key]);
}

export function useQueueNames(): (id: number | undefined) => string {
  const queues = useLookup('queues');
  return useMemo(() => {
    const m = new Map(queues.map((q) => [q.id, q.name]));
    return (id) => (!id ? '—' : m.get(id) ?? `Queue #${id}`);
  }, [queues]);
}

export function useUserNames(): (id: number | undefined) => string {
  const users = useLookup('users');
  const agents = useStore(live, (s) => s.agents);
  return useMemo(() => {
    const m = new Map<number, string>();
    for (const a of Object.values(agents)) m.set(a.user_id, a.display_name || a.username);
    for (const u of users) m.set(u.id, u.display_name || u.username);
    return (id) => (!id ? '—' : m.get(id) ?? `User #${id}`);
  }, [users, agents]);
}
