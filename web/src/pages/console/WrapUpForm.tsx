import { useState } from 'react';
import { ClipboardCheck } from 'lucide-react';
import type { Call } from '../../api/types';
import { patch } from '../../api/client';
import { dismissWrapup } from '../../state/live';
import { useLookup } from '../../state/lookups';
import { Card, Field } from '../../components/ui/primitives';
import { TagInput } from '../../components/ui/TagInput';
import { toast } from '../../components/ui/Toast';
import { callerLabel } from '../../lib/calls';
import { fmtDuration } from '../../lib/format';

/** After-call work: disposition, notes and tags saved to PATCH /calls/{id}. */
export function WrapUpForm({ call }: { call: Call }) {
  const dispositions = useLookup('dispositions').filter((d) => d.enabled);
  const [disposition, setDisposition] = useState(call.disposition || '');
  const [notes, setNotes] = useState(call.notes || '');
  const [tags, setTags] = useState<string[]>(call.tags ?? []);
  const [busy, setBusy] = useState(false);

  const save = async () => {
    setBusy(true);
    try {
      await patch(`/calls/${encodeURIComponent(call.id)}`, { disposition, notes, tags });
      toast.success('Wrap-up saved');
      dismissWrapup(call.id);
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };

  return (
    <Card title="Wrap-up" icon={<ClipboardCheck />} className="wrapup" actions={
      <span className="small muted">{callerLabel(call)} · talk {fmtDuration(call.talk_sec)}</span>
    }>
      <form className="col gap-lg" onSubmit={(e) => { e.preventDefault(); void save(); }}>
        <Field label="Disposition">
          <select className="select" value={disposition} onChange={(e) => setDisposition(e.target.value)} autoFocus>
            <option value="">Select a disposition…</option>
            {dispositions.map((d) => <option key={d.id} value={d.code}>{d.label}</option>)}
          </select>
        </Field>
        <Field label="Notes">
          <textarea className="textarea" value={notes} onChange={(e) => setNotes(e.target.value)} placeholder="What happened on this call?" />
        </Field>
        <div className="field">
          <span className="field-label">Tags</span>
          <TagInput value={tags} onChange={setTags} />
        </div>
        <div className="row end">
          <button type="button" className="btn ghost" onClick={() => dismissWrapup(call.id)}>Skip</button>
          <button type="submit" className="btn primary" disabled={busy || !disposition}>Save wrap-up</button>
        </div>
      </form>
    </Card>
  );
}
