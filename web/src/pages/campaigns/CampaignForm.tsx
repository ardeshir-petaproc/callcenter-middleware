import type { Campaign } from '../../api/types';
import { Field } from '../../components/ui/primitives';
import { useLookup } from '../../state/lookups';

export const blankCampaign: Campaign = {
  id: 0, name: '', description: '', queue_id: 0, status: 'draft', max_attempts: 3, retry_minutes: 30, schedule_id: 0,
  script: 'Hi, this is {agent} calling from Acme about …',
};

interface Props { c: Campaign; set: (p: Partial<Campaign>) => void; disabled: boolean }

export function CampaignForm({ c, set, disabled }: Props) {
  const queues = useLookup('queues');
  const schedules = useLookup('schedules');
  return (
    <>
      <div className="form-section">
        <div className="form-grid">
          <Field label="Name"><input className="input" value={c.name} disabled={disabled} onChange={(e) => set({ name: e.target.value })} /></Field>
          <Field label="Agent queue" hint="Available members of this queue make the calls">
            <select className="select" value={c.queue_id || ''} disabled={disabled} onChange={(e) => set({ queue_id: Number(e.target.value) || 0 })}>
              <option value="">Select a queue…</option>
              {queues.map((q) => <option key={q.id} value={q.id}>{q.name}</option>)}
            </select>
          </Field>
          <Field label="Description" className="span-all"><input className="input" value={c.description} disabled={disabled} onChange={(e) => set({ description: e.target.value })} /></Field>
          <Field label="Max attempts per lead"><input className="input" type="number" min={1} value={c.max_attempts} disabled={disabled} onChange={(e) => set({ max_attempts: Number(e.target.value) })} /></Field>
          <Field label="Retry after (minutes)"><input className="input" type="number" min={1} value={c.retry_minutes} disabled={disabled} onChange={(e) => set({ retry_minutes: Number(e.target.value) })} /></Field>
          <Field label="Dialing hours" hint="Only dial within this schedule">
            <select className="select" value={c.schedule_id || ''} disabled={disabled} onChange={(e) => set({ schedule_id: Number(e.target.value) || 0 })}>
              <option value="">Any time</option>
              {schedules.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
            </select>
          </Field>
        </div>
      </div>
      <div className="form-section">
        <Field label="Agent script" hint="Shown in the agent console during campaign calls. {agent} is replaced with the agent's name.">
          <textarea className="textarea" style={{ minHeight: 140 }} value={c.script} disabled={disabled} onChange={(e) => set({ script: e.target.value })} />
        </Field>
      </div>
    </>
  );
}
