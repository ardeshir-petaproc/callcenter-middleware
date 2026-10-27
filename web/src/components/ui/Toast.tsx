import { AlertCircle, CheckCircle2, Info, X } from 'lucide-react';
import { createStore, useStore } from '../../lib/store';
import { errMsg } from '../../api/client';

type Kind = 'success' | 'error' | 'info';
interface ToastItem { id: number; kind: Kind; text: string }

const store = createStore<{ items: ToastItem[] }>({ items: [] });
let seq = 0;

function push(kind: Kind, text: string, ms = kind === 'error' ? 6000 : 3500) {
  const id = ++seq;
  store.set((s) => ({ items: [...s.items.slice(-4), { id, kind, text }] }));
  window.setTimeout(() => dismiss(id), ms);
}

function dismiss(id: number) {
  store.set((s) => ({ items: s.items.filter((t) => t.id !== id) }));
}

export const toast = {
  success: (t: string) => push('success', t),
  error: (e: unknown) => push('error', typeof e === 'string' ? e : errMsg(e)),
  info: (t: string) => push('info', t),
};

const icons = { success: CheckCircle2, error: AlertCircle, info: Info };

export function Toaster() {
  const items = useStore(store, (s) => s.items);
  return (
    <div className="toasts" role="region" aria-live="polite" aria-label="Notifications">
      {items.map((t) => {
        const Icon = icons[t.kind];
        return (
          <div key={t.id} className={`toast ${t.kind}`} role={t.kind === 'error' ? 'alert' : 'status'}>
            <Icon aria-hidden />
            <div className="grow">{t.text}</div>
            <button className="close" onClick={() => dismiss(t.id)} aria-label="Dismiss"><X size={16} /></button>
          </div>
        );
      })}
    </div>
  );
}
