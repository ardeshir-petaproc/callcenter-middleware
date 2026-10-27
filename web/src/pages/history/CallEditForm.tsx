import { useEffect, useState } from 'react';
import { Save } from 'lucide-react';
import type { Call } from '../../api/types';
import { patch } from '../../api/client';
import { useLookup } from '../../state/lookups';
import { Card, Field } from '../../components/ui/primitives';
import { TagInput } from '../../components/ui/TagInput';
import { toast } from '../../components/ui/Toast';

export function CallEditForm({ call, onSaved }: { call: Call; onSaved: (c: Call) => void }) {
  const dispositions = useLookup('dispositions');
  const [disposition, setDisposition] = useState(call.disposition);
  const [notes, setNotes] = useState(call.notes);
  const [tags, setTags] = useState<string[]>(call.tags ?? []);
  const [busy, setBusy] = useState(false);
  useEffect(() => { setDisposition(call.disposition); setNotes(call.notes); setTags(call.tags ?? []); }, [call]);

  const dirty = disposition !== call.disposition || notes !== call.notes || JSON.stringify(tags) !== JSON.stringify(call.tags ?? []);
  const save = async () => {
    setBusy(true);
    try {
      const updated = await patch<Call>(`/calls/${encodeURIComponent(call.id)}`, { disposition, notes, tags });
      onSaved(updated && typeof updated === 'object' && 'id' in updated ? updated : { ...call, disposition, notes, tags });
      toast.success('Call updated');
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  const known = dispositions.some((d) => d.code === disposition);

  return (
    <Card title="Disposition & notes">
      <div className="col gap-lg">
        <Field label="Disposition">
          <select className="select" value={disposition} onChange={(e) => setDisposition(e.target.value)}>
            <option value="">None</option>
            {!known && disposition && <option value={disposition}>{disposition}</option>}
            {dispositions.map((d) => <option key={d.id} value={d.code}>{d.label}{d.enabled ? '' : ' (disabled)'}</option>)}
          </select>
        </Field>
        <Field label="Notes"><textarea className="textarea" value={notes} onChange={(e) => setNotes(e.target.value)} /></Field>
        <div className="field"><span className="field-label">Tags</span><TagInput value={tags} onChange={setTags} /></div>
        <div className="row end">
          <button className="btn primary" disabled={!dirty || busy} onClick={save}><Save />Save</button>
        </div>
      </div>
    </Card>
  );
}
