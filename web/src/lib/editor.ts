import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useDraft } from './hooks';
import type { useCrud } from './crud';
import { confirm } from '../components/ui/Confirm';
import { toast } from '../components/ui/Toast';

interface Opts<T extends { id: number }> {
  crud: ReturnType<typeof useCrud<T>>;
  id: string | undefined;
  blank: T;
  /** List route, e.g. "/org/departments". */ base: string;
  noun: string;
  label: (t: T) => string;
  validate?: (t: T) => string | null;
  /** Normalize a loaded entity (e.g. replace null arrays). */ normalize?: (t: T) => T;
}

/** Draft/save/delete wiring shared by master-detail configuration pages. */
export function useEntityEditor<T extends { id: number }>({ crud, id, blank, base, noun, label, validate, normalize }: Opts<T>) {
  const nav = useNavigate();
  const isNew = id === 'new';
  const source = useMemo(() => {
    if (isNew) return { ...blank };
    const found = crud.items.find((x) => String(x.id) === id);
    return found && normalize ? normalize(found) : found;
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isNew, id, crud.items]);
  const { draft, set, setDraft, dirty } = useDraft<T>(source, id ?? '');
  const [busy, setBusy] = useState(false);

  const save = async () => {
    if (!draft) return;
    const err = validate?.(draft);
    if (err) { toast.error(err); return; }
    setBusy(true);
    try {
      const saved = await crud.save(isNew ? ({ ...draft, id: undefined } as Partial<T>) : draft);
      toast.success(isNew ? `${noun} created` : `${noun} saved`);
      if (isNew) nav(`${base}/${saved.id}`, { replace: true });
      return saved;
    } catch (e) { toast.error(e); } finally { setBusy(false); }
    return undefined;
  };
  const remove = async () => {
    if (!draft) return;
    if (!(await confirm({ title: `Delete ${noun.toLowerCase()}?`, message: `Delete "${label(draft)}"? This cannot be undone.`, danger: true, confirmLabel: 'Delete' }))) return;
    try { await crud.remove(draft.id); toast.success(`${noun} deleted`); nav(base, { replace: true }); } catch (e) { toast.error(e); }
  };
  return { draft, set, setDraft, dirty, busy, isNew, save, remove };
}
