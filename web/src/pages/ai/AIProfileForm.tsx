import type { AIProfile, AIStatus } from '../../api/types';
import { Field, Toggle } from '../../components/ui/primitives';
import { ChipSelect } from '../../components/ui/TagInput';
import { KnowledgeScope } from './KnowledgeScope';
import { useLookup } from '../../state/lookups';
import { humanize } from '../../lib/format';

export const blankProfile: AIProfile = {
  id: 0, name: '', model: '', voice: '', language: 'en', instructions: '', greeting: '', temperature: 0.8,
  tools: [], handoff_queue_id: 0, max_duration_sec: 600, knowledge_base: '', enabled: true, kb_department_ids: [], kb_categories: [],
};

const TOOL_LABELS: Record<string, string> = {
  transfer_to_queue: 'Transfer to queue', end_call: 'End call', create_callback: 'Create callback', lookup_contact: 'Look up contact',
  take_message: 'Take a message', check_business_hours: 'Check business hours', route_call: 'Route call (routing rules)',
  search_knowledge: 'Search knowledge base', create_ticket: 'Create ticket', ticket_status: 'Ticket status', update_ticket: 'Update ticket',
};

interface Props { p: AIProfile; set: (x: Partial<AIProfile>) => void; disabled: boolean; status?: AIStatus }

export function AIProfileForm({ p, set, disabled, status }: Props) {
  const queues = useLookup('queues');
  const voices = status?.voices ?? [];
  const tools = status?.tools ?? [];
  return (
    <>
      <div className="form-section">
        <h3>Identity</h3>
        <div className="form-grid">
          <Field label="Name"><input className="input" value={p.name} disabled={disabled} onChange={(e) => set({ name: e.target.value })} /></Field>
          <Field label="Model" hint={status?.model ? `Server default: ${status.model}` : undefined}>
            <input className="input" value={p.model} placeholder={status?.model} disabled={disabled} onChange={(e) => set({ model: e.target.value })} />
          </Field>
          <Field label="Voice">
            <select className="select" value={p.voice} disabled={disabled} onChange={(e) => set({ voice: e.target.value })}>
              <option value="">Default{status?.voice ? ` (${status.voice})` : ''}</option>
              {voices.map((v) => <option key={v} value={v}>{humanize(v)}</option>)}
              {p.voice && !voices.includes(p.voice) && <option value={p.voice}>{p.voice}</option>}
            </select>
          </Field>
          <Field label="Language" hint="e.g. en, es, it"><input className="input" value={p.language} disabled={disabled} onChange={(e) => set({ language: e.target.value })} /></Field>
          <Field label={`Temperature (${p.temperature ? p.temperature.toFixed(1) : 'server default'})`} hint="0 uses the server default; OpenAI accepts 0.6–1.2">
            <input type="range" min={0} max={1.2} step={0.1} value={p.temperature} disabled={disabled} onChange={(e) => set({ temperature: Number(e.target.value) })} />
          </Field>
          <Field label="Max duration (s)"><input className="input" type="number" min={30} value={p.max_duration_sec} disabled={disabled} onChange={(e) => set({ max_duration_sec: Number(e.target.value) })} /></Field>
        </div>
      </div>
      <div className="form-section">
        <h3>Conversation</h3>
        <Field label="Greeting" hint="First sentence the agent says"><input className="input" value={p.greeting} disabled={disabled} onChange={(e) => set({ greeting: e.target.value })} /></Field>
        <Field label="Instructions" hint="System prompt: persona, goals, tone and rules">
          <textarea className="textarea" style={{ minHeight: 160 }} value={p.instructions} disabled={disabled} onChange={(e) => set({ instructions: e.target.value })} />
        </Field>
        <Field label="Inline knowledge" hint="Short facts always in the prompt: hours, prices, policies. Larger documents belong in the knowledge base.">
          <textarea className="textarea" style={{ minHeight: 140 }} value={p.knowledge_base} disabled={disabled} onChange={(e) => set({ knowledge_base: e.target.value })} />
        </Field>
      </div>
      <KnowledgeScope p={p} set={set} disabled={disabled} />
      <div className="form-section">
        <h3>Tools & hand-off</h3>
        {tools.length ? (
          <ChipSelect options={tools.map((t) => ({ value: t, label: TOOL_LABELS[t] ?? humanize(t) }))} value={p.tools ?? []} onChange={(v) => set({ tools: v })} disabled={disabled} />
        ) : <span className="small muted">No tools reported by the server.</span>}
        <div className="form-grid">
          <Field label="Hand-off queue" hint="Where the agent transfers when a human is needed">
            <select className="select" value={p.handoff_queue_id || ''} disabled={disabled} onChange={(e) => set({ handoff_queue_id: Number(e.target.value) || 0 })}>
              <option value="">Default queue</option>
              {queues.map((q) => <option key={q.id} value={q.id}>{q.name}</option>)}
            </select>
          </Field>
          <div className="field" style={{ justifyContent: 'flex-end' }}>
            <Toggle checked={p.enabled} disabled={disabled} onChange={(v) => set({ enabled: v })} label="Profile enabled" />
          </div>
        </div>
      </div>
    </>
  );
}
