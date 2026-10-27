import { useState } from 'react';
import { Check, Plus, Trash2 } from 'lucide-react';
import { useCrud } from '../../lib/crud';
import { Card, Empty, ErrorBox, SkeletonRows, Toggle } from '../../components/ui/primitives';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';

interface Col<T> { key: keyof T & string; label: string; placeholder?: string; width?: number }
type Lookup = 'dispositions' | 'pauseReasons';

interface Props<T extends { id: number; enabled: boolean }> {
  title: string;
  path: string;
  lookup: Lookup;
  cols: Col<T>[];
  blank: Omit<T, 'id'>;
  describe: (t: T) => string;
}

/** Small inline-editable table for simple lookup entities (dispositions, pause reasons). */
export function InlineCrud<T extends { id: number; enabled: boolean }>({ title, path, lookup, cols, blank, describe }: Props<T>) {
  const crud = useCrud<T>(path, lookup);
  const [edits, setEdits] = useState<Record<number, Partial<T>>>({});
  const [draft, setDraft] = useState<Omit<T, 'id'>>(blank);

  const row = (t: T): T => ({ ...t, ...edits[t.id] });
  const change = (id: number, p: Partial<T>) => setEdits((e) => ({ ...e, [id]: { ...e[id], ...p } }));
  const save = async (t: T) => {
    try {
      await crud.save(row(t));
      setEdits((e) => { const n = { ...e }; delete n[t.id]; return n; });
      toast.success('Saved');
    } catch (e) { toast.error(e); }
  };
  const toggle = async (t: T, enabled: boolean) => {
    try { await crud.save({ ...row(t), enabled }); } catch (e) { toast.error(e); }
  };
  const add = async () => {
    try { await crud.save(draft as Partial<T>); setDraft(blank); toast.success('Added'); } catch (e) { toast.error(e); }
  };
  const remove = async (t: T) => {
    if (!(await confirm({ title: `Delete ${describe(t)}?`, message: 'Historic calls keep their recorded value.', danger: true, confirmLabel: 'Delete' }))) return;
    try { await crud.remove(t.id); } catch (e) { toast.error(e); }
  };
  const canAdd = cols.every((c) => String((draft as Record<string, unknown>)[c.key] ?? '').trim());

  return (
    <Card title={title} flush>
      {crud.error ? <div style={{ padding: 16 }}><ErrorBox error={crud.error} onRetry={crud.reload} /></div>
        : crud.loading && !crud.data ? <SkeletonRows rows={4} />
        : (
          <div className="table-wrap">
            <table className="table">
              <thead><tr>{cols.map((c) => <th key={c.key}>{c.label}</th>)}<th>Enabled</th><th /></tr></thead>
              <tbody>
                {crud.items.length === 0 && <tr><td colSpan={cols.length + 2}><Empty title="None defined yet" /></td></tr>}
                {crud.items.map((t) => {
                  const r = row(t);
                  const dirty = !!edits[t.id];
                  return (
                    <tr key={t.id}>
                      {cols.map((c) => (
                        <td key={c.key} style={c.width ? { width: c.width } : undefined}>
                          <input className="input sm" value={String(r[c.key] ?? '')} aria-label={c.label}
                            onChange={(e) => change(t.id, { [c.key]: e.target.value } as Partial<T>)}
                            onKeyDown={(e) => { if (e.key === 'Enter' && dirty) void save(t); }} />
                        </td>
                      ))}
                      <td><Toggle checked={r.enabled} onChange={(v) => toggle(t, v)} /></td>
                      <td className="actions">
                        {dirty && <button className="btn sm primary icon" aria-label="Save" onClick={() => save(t)}><Check /></button>}
                        <button className="btn sm ghost icon" aria-label="Delete" onClick={() => remove(t)}><Trash2 /></button>
                      </td>
                    </tr>
                  );
                })}
                <tr>
                  {cols.map((c) => (
                    <td key={c.key}>
                      <input className="input sm" placeholder={c.placeholder ?? c.label} value={String((draft as Record<string, unknown>)[c.key] ?? '')} aria-label={`New ${c.label}`}
                        onChange={(e) => setDraft((d) => ({ ...d, [c.key]: e.target.value }))}
                        onKeyDown={(e) => { if (e.key === 'Enter' && canAdd) void add(); }} />
                    </td>
                  ))}
                  <td />
                  <td className="actions"><button className="btn sm" disabled={!canAdd} onClick={add}><Plus />Add</button></td>
                </tr>
              </tbody>
            </table>
          </div>
        )}
    </Card>
  );
}
