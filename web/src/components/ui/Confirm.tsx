import { useState } from 'react';
import { createStore, useStore } from '../../lib/store';
import { Dialog } from './Dialog';

interface ConfirmReq {
  title: string;
  message: string;
  confirmLabel?: string;
  danger?: boolean;
  resolve: (ok: boolean) => void;
}

const store = createStore<{ req: ConfirmReq | null }>({ req: null });

/** Promise-based confirmation dialog: `if (await confirm({...})) ...` */
export function confirm(opts: Omit<ConfirmReq, 'resolve'>): Promise<boolean> {
  return new Promise((resolve) => store.set({ req: { ...opts, resolve } }));
}

export function ConfirmHost() {
  const req = useStore(store, (s) => s.req);
  const [busy, setBusy] = useState(false);
  const close = (ok: boolean) => {
    req?.resolve(ok);
    setBusy(false);
    store.set({ req: null });
  };
  return (
    <Dialog
      open={!!req}
      size="narrow"
      title={req?.title ?? ''}
      onClose={() => close(false)}
      footer={
        <>
          <button className="btn" onClick={() => close(false)}>Cancel</button>
          <button
            className={`btn ${req?.danger ? 'danger' : 'primary'}`}
            disabled={busy}
            data-autofocus
            onClick={() => { setBusy(true); close(true); }}
          >
            {req?.confirmLabel ?? 'Confirm'}
          </button>
        </>
      }
    >
      <p className="pre">{req?.message}</p>
    </Dialog>
  );
}
