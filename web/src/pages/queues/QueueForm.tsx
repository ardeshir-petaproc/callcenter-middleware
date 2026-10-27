import type { Queue } from '../../api/types';
import { Field, Toggle } from '../../components/ui/primitives';
import { TagInput } from '../../components/ui/TagInput';
import { ActionEditor } from '../../components/ActionEditor';

const STRATEGIES: { value: Queue['strategy']; label: string; hint: string }[] = [
  { value: 'longest_idle', label: 'Longest idle', hint: 'Agent idle the longest gets the call' },
  { value: 'ringall', label: 'Ring all', hint: 'Ring every available agent at once' },
  { value: 'round_robin', label: 'Round robin', hint: 'Rotate through agents in order' },
  { value: 'fewest_calls', label: 'Fewest calls', hint: 'Agent with fewest calls today' },
  { value: 'random', label: 'Random', hint: 'Pick a random available agent' },
];

export const blankQueue: Omit<Queue, 'id'> & { id: number } = {
  id: 0, name: '', description: '', mode: 'native', extension: '', strategy: 'longest_idle', priority: 0,
  ring_timeout_sec: 20, max_wait_sec: 300, wrapup_sec: 15, sla_sec: 20, moh_class: 'default', join_media: '',
  required_skills: [], record: false, allow_callback: true, leave_when_empty: false, survey: false, alert_waiting: 0, alert_wait_sec: 0, overflow: { type: 'voicemail' }, enabled: true,
};

interface Props { q: Queue; set: (p: Partial<Queue>) => void; disabled: boolean }

function Num({ label, value, onChange, disabled, hint, min = 0 }: { label: string; value: number; onChange: (n: number) => void; disabled: boolean; hint?: string; min?: number }) {
  return (
    <Field label={label} hint={hint}>
      <input className="input" type="number" min={min} value={Number.isFinite(value) ? value : 0} disabled={disabled} onChange={(e) => onChange(Number(e.target.value))} />
    </Field>
  );
}

export function QueueForm({ q, set, disabled }: Props) {
  const strategy = STRATEGIES.find((s) => s.value === q.strategy);
  return (
    <>
      <div className="form-section">
        <h3>General</h3>
        <div className="form-grid">
          <Field label="Name"><input className="input" value={q.name} disabled={disabled} onChange={(e) => set({ name: e.target.value })} required /></Field>
          <Field label="Extension" hint="Optional direct-dial number"><input className="input" value={q.extension} disabled={disabled} onChange={(e) => set({ extension: e.target.value })} /></Field>
          <Field label="Description" className="span-all"><input className="input" value={q.description} disabled={disabled} onChange={(e) => set({ description: e.target.value })} /></Field>
          <Field label="Mode" hint={q.mode === 'native' ? 'Routing by this middleware (ACD)' : 'Delegated to Asterisk app_queue'}>
            <select className="select" value={q.mode} disabled={disabled} onChange={(e) => set({ mode: e.target.value as Queue['mode'] })}>
              <option value="native">Native</option><option value="asterisk">Asterisk</option>
            </select>
          </Field>
          <Field label="Strategy" hint={strategy?.hint}>
            <select className="select" value={q.strategy} disabled={disabled} onChange={(e) => set({ strategy: e.target.value as Queue['strategy'] })}>
              {STRATEGIES.map((s) => <option key={s.value} value={s.value}>{s.label}</option>)}
            </select>
          </Field>
          <Num label="Priority" hint="Higher is served first" value={q.priority} disabled={disabled} onChange={(n) => set({ priority: n })} />
        </div>
      </div>
      <div className="form-section">
        <h3>Timing & service level</h3>
        <div className="form-grid">
          <Num label="Ring timeout (s)" value={q.ring_timeout_sec} disabled={disabled} onChange={(n) => set({ ring_timeout_sec: n })} min={1} />
          <Num label="Max wait (s)" hint="Then the overflow action runs" value={q.max_wait_sec} disabled={disabled} onChange={(n) => set({ max_wait_sec: n })} />
          <Num label="Wrap-up (s)" value={q.wrapup_sec} disabled={disabled} onChange={(n) => set({ wrapup_sec: n })} />
          <Num label="SLA target (s)" hint="Answered within this counts as met" value={q.sla_sec} disabled={disabled} onChange={(n) => set({ sla_sec: n })} />
        </div>
      </div>
      <div className="form-section">
        <h3>Media & skills</h3>
        <div className="form-grid">
          <Field label="Music on hold class"><input className="input" value={q.moh_class} disabled={disabled} onChange={(e) => set({ moh_class: e.target.value })} /></Field>
          <Field label="Join announcement" hint="e.g. sound:custom/please-hold"><input className="input" value={q.join_media} disabled={disabled} onChange={(e) => set({ join_media: e.target.value })} /></Field>
          <div className="field span-all">
            <span className="field-label">Required skills</span>
            <TagInput value={q.required_skills ?? []} onChange={(v) => set({ required_skills: v })} disabled={disabled} placeholder="e.g. spanish, billing" />
          </div>
        </div>
      </div>
      <div className="form-section">
        <h3>Supervisor alerts</h3>
        <p className="small muted">Supervisors get a live alert (and webhooks fire <code>queue.alert</code>) when either threshold is reached. 0 turns a threshold off.</p>
        <div className="form-grid">
          <Num label="Alert when callers waiting ≥" value={q.alert_waiting ?? 0} disabled={disabled} onChange={(n) => set({ alert_waiting: n })} />
          <Num label="Alert when longest wait ≥ (s)" value={q.alert_wait_sec ?? 0} disabled={disabled} onChange={(n) => set({ alert_wait_sec: n })} />
        </div>
      </div>
      <div className="form-section">
        <h3>Overflow</h3>
        <p className="small muted">What happens when the max wait is exceeded{q.leave_when_empty ? ' or no agents are logged in' : ''}.</p>
        <ActionEditor value={q.overflow} onChange={(a) => set({ overflow: a })} disabled={disabled} />
      </div>
      <div className="form-section">
        <h3>Options</h3>
        <div className="col">
          <Toggle checked={q.enabled} disabled={disabled} onChange={(v) => set({ enabled: v })} label="Queue enabled" />
          <Toggle checked={q.record} disabled={disabled} onChange={(v) => set({ record: v })} label="Record calls" />
          <Toggle checked={q.allow_callback} disabled={disabled} onChange={(v) => set({ allow_callback: v })} label="Allow callers to request a callback" />
          <Toggle checked={!!q.survey} disabled={disabled} onChange={(v) => set({ survey: v })} label="Post-call satisfaction survey (caller rates 1–5 on the keypad)" />
          <Toggle checked={q.leave_when_empty} disabled={disabled} onChange={(v) => set({ leave_when_empty: v })} label="Overflow immediately when no agents are logged in" />
        </div>
      </div>
    </>
  );
}
