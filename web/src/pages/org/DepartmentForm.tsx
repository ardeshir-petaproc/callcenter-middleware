import type { Department } from '../../api/types_org';
import { Field, Toggle } from '../../components/ui/primitives';
import { TagInput } from '../../components/ui/TagInput';
import { ActionEditor } from '../../components/ActionEditor';
import { useLookup } from '../../state/lookups';

interface Props { d: Department; set: (p: Partial<Department>) => void; disabled: boolean }

export function DepartmentForm({ d, set, disabled }: Props) {
  const users = useLookup('users');
  const schedules = useLookup('schedules');
  return (
    <>
      <div className="form-section">
        <h3>General</h3>
        <div className="form-grid">
          <Field label="Name"><input className="input" dir="auto" value={d.name} disabled={disabled} onChange={(e) => set({ name: e.target.value })} /></Field>
          <Field label="Manager">
            <select className="select" value={d.manager_id || ''} disabled={disabled} onChange={(e) => set({ manager_id: Number(e.target.value) || 0 })}>
              <option value="">No manager</option>
              {users.map((u) => <option key={u.id} value={u.id}>{u.display_name || u.username}</option>)}
            </select>
          </Field>
          <Field label="Description" className="span-all"><input className="input" dir="auto" value={d.description} disabled={disabled} onChange={(e) => set({ description: e.target.value })} /></Field>
        </div>
      </div>
      <div className="form-section">
        <h3>Availability</h3>
        <div className="form-grid">
          <Field label="Opening hours" hint="Outside these hours the fallback runs">
            <select className="select" value={d.schedule_id || ''} disabled={disabled} onChange={(e) => set({ schedule_id: Number(e.target.value) || 0 })}>
              <option value="">Always open</option>
              {schedules.map((s) => <option key={s.id} value={s.id}>{s.name}</option>)}
            </select>
          </Field>
          <Field label="Capacity" hint="Max concurrent calls, 0 = unlimited">
            <input className="input" type="number" min={0} value={d.capacity} disabled={disabled} onChange={(e) => set({ capacity: Number(e.target.value) })} />
          </Field>
          <Field label="Priority" hint="Higher is preferred when several departments match">
            <input className="input" type="number" value={d.priority} disabled={disabled} onChange={(e) => set({ priority: Number(e.target.value) })} />
          </Field>
          <div className="field span-all">
            <span className="field-label">Languages</span>
            <TagInput value={d.languages} onChange={(v) => set({ languages: v })} disabled={disabled} placeholder="e.g. en, fa" suggestions={['en', 'fa', 'ar', 'tr', 'de', 'fr']} />
          </div>
        </div>
      </div>
      <div className="form-section">
        <h3>Fallback</h3>
        <p className="small muted">What happens when the department is closed, over capacity or nobody is available.</p>
        <ActionEditor value={d.fallback} onChange={(a) => set({ fallback: a })} disabled={disabled} />
      </div>
      <div className="form-section">
        <Toggle checked={d.enabled} disabled={disabled} onChange={(v) => set({ enabled: v })} label="Department enabled" />
      </div>
    </>
  );
}
