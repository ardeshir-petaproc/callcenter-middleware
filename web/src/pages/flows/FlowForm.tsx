import type { IVRFlow } from '../../api/types';
import { Field, Toggle } from '../../components/ui/primitives';
import { ActionEditor } from '../../components/ActionEditor';
import { useLookup } from '../../state/lookups';
import { KeypadEditor } from './KeypadEditor';

export const blankFlow: IVRFlow = {
  id: 0, name: '', dnis: '', greeting_media: 'sound:hello-world', invalid_media: 'sound:option-is-invalid', timeout_sec: 5,
  max_retries: 3, options: {}, timeout_action: { type: '' }, direct_action: { type: '' }, schedule_id: 0,
  after_hours_action: { type: 'voicemail' }, enabled: true,
};

interface Props { f: IVRFlow; set: (p: Partial<IVRFlow>) => void; disabled: boolean }

export function FlowForm({ f, set, disabled }: Props) {
  const schedules = useLookup('schedules');
  return (
    <>
      <div className="form-section">
        <h3>General</h3>
        <div className="form-grid">
          <Field label="Name"><input className="input" value={f.name} disabled={disabled} onChange={(e) => set({ name: e.target.value })} /></Field>
          <Field label="DNIS" hint="Dialed number that enters this flow"><input className="input" value={f.dnis} disabled={disabled} onChange={(e) => set({ dnis: e.target.value })} /></Field>
          <Field label="Greeting media" hint="e.g. sound:custom/welcome"><input className="input" value={f.greeting_media} disabled={disabled} onChange={(e) => set({ greeting_media: e.target.value })} /></Field>
          <Field label="Invalid-input media"><input className="input" value={f.invalid_media} disabled={disabled} onChange={(e) => set({ invalid_media: e.target.value })} /></Field>
          <Field label="Input timeout (s)"><input className="input" type="number" min={1} value={f.timeout_sec} disabled={disabled} onChange={(e) => set({ timeout_sec: Number(e.target.value) })} /></Field>
          <Field label="Max retries"><input className="input" type="number" min={0} value={f.max_retries} disabled={disabled} onChange={(e) => set({ max_retries: Number(e.target.value) })} /></Field>
          <div className="field span-all"><Toggle checked={f.enabled} disabled={disabled} onChange={(v) => set({ enabled: v })} label="Flow enabled" /></div>
        </div>
      </div>
      <div className="form-section">
        <h3>Skip menu</h3>
        <p className="small muted">Set a direct action to route every caller immediately after the greeting, without a menu.</p>
        <ActionEditor value={f.direct_action} disabled={disabled} excludeFlowId={f.id} onChange={(a) => set({ direct_action: a })} />
      </div>
      {!f.direct_action?.type && (
        <>
          <div className="form-section">
            <h3>Keypad options</h3>
            <KeypadEditor options={f.options ?? {}} disabled={disabled} flowId={f.id} onChange={(o) => set({ options: o })} />
          </div>
          <div className="form-section">
            <h3>No input</h3>
            <p className="small muted">Runs after {f.timeout_sec}s of silence on the last retry.</p>
            <ActionEditor value={f.timeout_action} disabled={disabled} excludeFlowId={f.id} onChange={(a) => set({ timeout_action: a })} />
          </div>
        </>
      )}
      <div className="form-section">
        <h3>Business hours</h3>
        <div className="form-grid">
          <Field label="Schedule" hint="Callers outside these hours get the after-hours action">
            <select className="select" value={f.schedule_id || ''} disabled={disabled} onChange={(e) => set({ schedule_id: Number(e.target.value) || 0 })}>
              <option value="">Always open</option>
              {schedules.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
            </select>
          </Field>
        </div>
        {!!f.schedule_id && (
          <div className="field">
            <span className="field-label">After-hours action</span>
            <ActionEditor value={f.after_hours_action} disabled={disabled} excludeFlowId={f.id} onChange={(a) => set({ after_hours_action: a })} />
          </div>
        )}
      </div>
    </>
  );
}
