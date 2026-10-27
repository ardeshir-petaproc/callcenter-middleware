import { useEffect, useState } from 'react';
import { Plus, Save, Trash2 } from 'lucide-react';
import { get, put } from '../../api/client';
import { useApi } from '../../lib/hooks';
import { Card, ErrorBox, Field, SkeletonRows, Toggle } from '../../components/ui/primitives';
import { toast } from '../../components/ui/Toast';

type Settings = Record<string, string>;
const KNOWN = ['company_name', 'record_outbound'];

export function GeneralTab() {
  const { data, loading, error, reload, setData } = useApi((s) => get<Settings>('/settings', undefined, s), []);
  const [draft, setDraft] = useState<Settings>({});
  const [newKey, setNewKey] = useState('');
  const [busy, setBusy] = useState(false);
  useEffect(() => { if (data) setDraft({ ...data }); }, [data]);
  const dirty = JSON.stringify(draft) !== JSON.stringify(data ?? {});
  const set = (k: string, v: string) => setDraft((d) => ({ ...d, [k]: v }));

  const save = async () => {
    setBusy(true);
    try {
      const merged = await put<Settings>('/settings', draft);
      setData(merged && typeof merged === 'object' ? merged : draft);
      toast.success('Settings saved');
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };

  if (error) return <ErrorBox error={error} onRetry={reload} />;
  if (loading && !data) return <div className="card"><SkeletonRows rows={4} /></div>;
  const extra = Object.keys(draft).filter((k) => !KNOWN.includes(k)).sort();
  return (
    <Card title="General settings" flush>
      <div className="card-body col gap-lg">
        <Field label="Company name" hint="Shown in greetings and AI prompts where referenced">
          <input className="input" value={draft.company_name ?? ''} onChange={(e) => set('company_name', e.target.value)} />
        </Field>
        <Toggle checked={draft.record_outbound === 'true'} onChange={(v) => set('record_outbound', v ? 'true' : 'false')} label="Record outbound (click-to-call and callback) calls" />
        <div className="form-section">
          <h3>Advanced key/value settings</h3>
          {extra.length === 0 && <div className="small muted">No other keys.</div>}
          {extra.map((k) => (
            <div key={k} className="row">
              <code style={{ minWidth: 160 }}>{k}</code>
              <input className="input sm grow" value={draft[k]} onChange={(e) => set(k, e.target.value)} aria-label={k} />
              <button className="btn ghost icon sm" aria-label={`Clear ${k}`} title="Clear value" onClick={() => set(k, '')}><Trash2 /></button>
            </div>
          ))}
          <div className="row">
            <input className="input sm" style={{ maxWidth: 220 }} placeholder="new_key" value={newKey} onChange={(e) => setNewKey(e.target.value.replace(/\s/g, '_'))} aria-label="New key" />
            <button className="btn sm" disabled={!newKey || newKey in draft} onClick={() => { set(newKey, ''); setNewKey(''); }}><Plus />Add key</button>
          </div>
        </div>
      </div>
      <div className="card-foot"><button className="btn primary" disabled={!dirty || busy} onClick={save}><Save />Save settings</button></div>
    </Card>
  );
}
