import type { JobProfile } from '../../api/types_org';
import { Field, Toggle } from '../../components/ui/primitives';
import { TagInput } from '../../components/ui/TagInput';
import { useLookup } from '../../state/lookups';

interface Props { j: JobProfile; set: (p: Partial<JobProfile>) => void; disabled: boolean }

export function JobProfileForm({ j, set, disabled }: Props) {
  const depts = useLookup('departments');
  const users = useLookup('users');
  const topics = useLookup('topics');
  return (
    <>
      <div className="form-section">
        <h3>Role</h3>
        <div className="form-grid">
          <Field label="Title"><input className="input" dir="auto" value={j.title} disabled={disabled} onChange={(e) => set({ title: e.target.value })} /></Field>
          <Field label="Department">
            <select className="select" value={j.department_id || ''} disabled={disabled} onChange={(e) => set({ department_id: Number(e.target.value) || 0 })}>
              <option value="">No department</option>
              {depts.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
            </select>
          </Field>
          <Field label="Description" className="span-all">
            <textarea className="textarea" dir="auto" value={j.description} disabled={disabled} onChange={(e) => set({ description: e.target.value })} />
          </Field>
          <div className="field span-all">
            <span className="field-label">Responsibilities</span>
            <TagInput value={j.responsibilities} onChange={(v) => set({ responsibilities: v })} disabled={disabled} placeholder="Type a responsibility and press Enter" />
          </div>
          <div className="field span-all">
            <span className="field-label">Request types handled</span>
            <TagInput value={j.request_types} onChange={(v) => set({ request_types: v })} disabled={disabled}
              placeholder="e.g. refund, billing" suggestions={topics.map((t) => t.name)} />
            <span className="hint">Routing topics or request keys this role can take.</span>
          </div>
        </div>
      </div>
      <div className="form-section">
        <h3>Authority & escalation</h3>
        <Field label="Authority limits" hint="What the holder may decide alone, e.g. refunds up to an amount">
          <textarea className="textarea" dir="auto" value={j.authority_limits} disabled={disabled} onChange={(e) => set({ authority_limits: e.target.value })} />
        </Field>
        <Field label="Escalation conditions">
          <textarea className="textarea" dir="auto" value={j.escalation_conditions} disabled={disabled} onChange={(e) => set({ escalation_conditions: e.target.value })} />
        </Field>
        <div className="form-grid">
          <Field label="Escalate to">
            <select className="select" value={j.escalate_to_user_id || ''} disabled={disabled} onChange={(e) => set({ escalate_to_user_id: Number(e.target.value) || 0 })}>
              <option value="">Nobody specific</option>
              {users.map((u) => <option key={u.id} value={u.id}>{u.display_name || u.username}</option>)}
            </select>
          </Field>
          <div className="field" style={{ justifyContent: 'flex-end' }}>
            <Toggle checked={j.is_template} disabled={disabled} onChange={(v) => set({ is_template: v })} label="Use as a template" />
          </div>
        </div>
      </div>
    </>
  );
}
