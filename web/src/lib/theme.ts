import { useSyncExternalStore } from 'react';

export type ThemePref = 'light' | 'dark' | 'system';
const KEY = 'cc.theme';
const subs = new Set<() => void>();

function read(): ThemePref {
  try {
    const v = localStorage.getItem(KEY);
    return v === 'light' || v === 'dark' ? v : 'system';
  } catch { return 'system'; }
}

let pref: ThemePref = read();

export function setTheme(p: ThemePref) {
  pref = p;
  try {
    if (p === 'system') localStorage.removeItem(KEY);
    else localStorage.setItem(KEY, p);
  } catch { /* ignore */ }
  if (p === 'system') delete document.documentElement.dataset.theme;
  else document.documentElement.dataset.theme = p;
  subs.forEach((fn) => fn());
}

export function effectiveTheme(): 'light' | 'dark' {
  if (pref !== 'system') return pref;
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

export function useTheme(): [ThemePref, (p: ThemePref) => void] {
  const p = useSyncExternalStore((fn) => { subs.add(fn); return () => subs.delete(fn); }, () => pref);
  return [p, setTheme];
}

export function cycleTheme() {
  const eff = effectiveTheme();
  setTheme(eff === 'dark' ? 'light' : 'dark');
}
