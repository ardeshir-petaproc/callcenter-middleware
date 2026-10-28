import { useEffect, useState } from 'react';
import { AlertTriangle } from 'lucide-react';
import { post, put } from '../../api/client';
import type { RoutingRule, TargetType } from '../../api/types_routing';
import { useLookup } from '../../state/lookups';
import { ActionEditor } from '../../components/ActionEditor';
import { Dialog } from '../../components/ui/Dialog';
import { Field, Toggle } from '../../components/ui/primitives';
import { ChipSelect, TagInput } from '../../components/ui/TagInput';
import { toast } from '../../components/ui/Toast';
import { LANG_SUGGESTIONS } from '../users/UserProfileFields';
import { TARGET_TYPES, TargetSelect } from './routingUi';

type Draft = Omit<RoutingRule, 'id' | 'status' | 'source' | 'version' | 'created_by' | 'approved_by' | 'approved_at' | 'updated_at'> & { id?: number };
const blank: Draft = {
  name: '', topics: [], keywords: [], languages: [], vip_only: false, target_type: 'job_profile', target_id: 0,
  fallback: { type: '' }, priority: 1, rationale: '', job_profile_id: 0,
};

/** Create or edit a routing rule. Saving always produces a draft that needs approval. */
export function RuleEditor({ open, rule, onClose, onSaved }: { open: boolean; rule: RoutingRule | null; onClose: () => void; onSaved: (r: RoutingRule) => void }) {
  const [r, setR] = useState<Draft>(blank);
  const [busy, setBusy] = useState(false);
  const topics = useLookup('topics');
  useEffect(() => {
    if (!open) return;
    setR(rule ? { ...rule, topics: rule.topics ?? [], keywords: rule.keywords ?? [], languages: rule.languages ?? [], fallback: rule.fallback ?? { type: '' } } : blank);
  }, [open, rule]);
  const set = (p: Partial<Draft>) => setR((x) => ({ ...x, ...p }));
  const wasApproved = rule?.status === 'approved';

  const save = async () => {
    setBusy(true);
    try {
      const saved = rule ? await put<RoutingRule>(`/routing/rules/${rule.id}`, { ...rule, ...r }) : await post<RoutingRule>('/routing/rules', r);
      toast.success(wasApproved ? 'Saved as draft — the rule must be approved again' : 'Rule saved as draft');
      onSaved(saved);
      onClose();
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };

  return (
    <Dialog open={open} onClose={onClose} size="wide" title={rule ? `Edit rule “${rule.name}”` : 'New routing rule'}
      footer={<><button className="btn" onClick={onClose}>Cancel</button>
        <button className="btn primary" disabled={busy || !r.name.trim() || !r.target_id} onClick={save}>{wasApproved ? 'Save as draft (needs re-approval)' : 'Save draft'}</button></>}>
      {wasApproved && <div className="alert warn"><AlertTriangle /><div>This rule is approved and routing calls. Saving changes returns it to <strong>draft</strong>: it stops routing until approved again.</div></div>}
      <div className="form-grid">
        <Field label="Name" className="span-2"><input className="input" dir="auto" value={r.name} onChange={(e) => set({ name: e.target.value })} data-autofocus /></Field>
        <Field label="Priority" hint="Higher wins when several rules match"><input className="input" type="number" value={r.priority} onChange={(e) => set({ priority: Number(e.target.value) })} /></Field>
      </div>
      <div className="form-section">
        <h3>When the request matches</h3>
        <div className="field"><span className="field-label">Topics (any)</span>
          {topics.length ? <ChipSelect options={topics.map((t) => ({ value: t.name, label: t.name }))} value={r.topics ?? []} onChange={(v) => set({ topics: v })} />
            : <span className="small muted">No topics defined yet.</span>}</div>
        <div className="field"><span className="field-label">Keywords (any)</span>
          <TagInput value={r.keywords ?? []} onChange={(v) => set({ keywords: v })} placeholder="e.g. chargeback, شکایت" /></div>
        <div className="form-grid">
          <div className="field"><span className="field-label">Languages</span>
            <TagInput value={r.languages ?? []} onChange={(v) => set({ languages: v })} placeholder="Any language" suggestions={LANG_SUGGESTIONS} /></div>
          <div className="field" style={{ justifyContent: 'flex-end' }}><Toggle checked={r.vip_only} onChange={(v) => set({ vip_only: v })} label="VIP callers only" /></div>
        </div>
      </div>
      <div className="form-section">
        <h3>Send it to</h3>
        <div className="form-grid">
          <Field label="Target type">
            <select className="select" value={r.target_type} onChange={(e) => set({ target_type: e.target.value as TargetType, target_id: 0 })}>
              {TARGET_TYPES.map((t) => <option key={t.value} value={t.value}>{t.label}</option>)}
            </select>
          </Field>
          <Field label="Target"><TargetSelect type={r.target_type} value={r.target_id} onChange={(id) => set({ target_id: id })} /></Field>
        </div>
        <div className="field"><span className="field-label">Fallback when the target cannot take the call</span>
          <ActionEditor value={r.fallback} onChange={(a) => set({ fallback: a })} allowMedia={false} /></div>
        <Field label="Rationale" hint="Why this rule exists — shown in routing explanations">
          <textarea className="textarea" dir="auto" value={r.rationale} onChange={(e) => set({ rationale: e.target.value })} />
        </Field>
      </div>
    </Dialog>
  );
}
