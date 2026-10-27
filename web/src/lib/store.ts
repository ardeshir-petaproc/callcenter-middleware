import { useSyncExternalStore } from 'react';

// A tiny zustand-like store built on useSyncExternalStore.
export interface Store<S> {
  get: () => S;
  set: (patch: Partial<S> | ((s: S) => Partial<S>)) => void;
  subscribe: (fn: () => void) => () => void;
}

export function createStore<S extends object>(initial: S): Store<S> {
  let state = initial;
  const subs = new Set<() => void>();
  return {
    get: () => state,
    set(patch) {
      const next = typeof patch === 'function' ? patch(state) : patch;
      state = { ...state, ...next };
      subs.forEach((fn) => fn());
    },
    subscribe(fn) {
      subs.add(fn);
      return () => { subs.delete(fn); };
    },
  };
}

/** Subscribe to a slice. The selector must return a stable reference (a field, or a primitive). */
export function useStore<S extends object, T>(store: Store<S>, selector: (s: S) => T): T {
  return useSyncExternalStore(store.subscribe, () => selector(store.get()), () => selector(store.get()));
}

// Shared one-second clock so every live timer ticks in sync with a single interval.
let now = Date.now();
let offset = 0; // server clock - local clock (ms)
const clockSubs = new Set<() => void>();
let timer: number | undefined;

function ensureTimer() {
  if (timer !== undefined) return;
  now = Date.now();
  timer = window.setInterval(() => {
    now = Date.now();
    clockSubs.forEach((fn) => fn());
  }, 1000);
}

export function setServerOffset(ms: number) {
  // ignore tiny jitter
  if (Math.abs(ms - offset) > 750) offset = ms;
}

export function serverNow(): number { return Date.now() + offset; }

/** Server-adjusted epoch ms, re-rendering every second. */
export function useNow(): number {
  return useSyncExternalStore(
    (fn) => {
      clockSubs.add(fn);
      ensureTimer();
      return () => {
        clockSubs.delete(fn);
        if (clockSubs.size === 0 && timer !== undefined) { clearInterval(timer); timer = undefined; }
      };
    },
    () => now + offset,
  );
}
