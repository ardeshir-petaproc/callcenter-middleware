import type { KnowledgeSource } from '../../api/types_work';
import { useLookup } from '../../state/lookups';
import { Field, Segmented, Toggle } from '../../components/ui/primitives';
import { ChipSelect, TagInput } from '../../components/ui/TagInput';

export type KBMeta = Pick<KnowledgeSource, 'title' | 'category' | 'tags' | 'department_ids' | 'visibility' | 'allowed_roles' | 'ai_enabled' | 'status' | 'valid_until' | 'topics'>;

/** Metadata of a knowledge source: classification, scope, visibility, AI use, status and validity. */
export function KBMetaFields({ m, set, categories }: { m: KBMeta; set: (p: Partial<KBMeta>) => void; categories: string[] }) {
  const depts = useLookup('departments');
  const roles = useLookup('customRoles');
  const topics = useLookup('topics');
  const roleOpts = [
    { value: 'admin', label: 'Admins' }, { value: 'supervisor', label: 'Supervisors' }, { value: 'agent', label: 'Agents' },
    ...roles.map((r) => ({ value: `role:${r.id}`, label: r.name })),
  ];
  return (
    <>
      <div className="form-grid">
        <Field label="Title" className="span-2"><input className="input" dir="auto" value={m.title} onChange={(e) => set({ title: e.target.value })} /></Field>
        <Field label="Category">
          <input className="input" dir="auto" list="kb-categories" value={m.category} onChange={(e) => set({ category: e.target.value })} />
          <datalist id="kb-categories">{categories.map((c) => <option key={c} value={c} />)}</datalist>
        </Field>
        <Field label="Status">
          <select className="select" value={m.status} onChange={(e) => set({ status: e.target.value as KBMeta['status'] })}>
            <option value="active">Active</option><option value="draft">Draft</option><option value="archived">Archived</option>
          </select>
        </Field>
        <Field label="Valid until" hint="Expired sources are hidden from the AI"><input className="input" type="date" value={m.valid_until} onChange={(e) => set({ valid_until: e.target.value })} /></Field>
        <div className="field span-all"><span className="field-label">Tags</span><TagInput value={m.tags ?? []} onChange={(v) => set({ tags: v })} /></div>
        <div className="field span-all"><span className="field-label">Topics</span>
          <TagInput value={m.topics ?? []} onChange={(v) => set({ topics: v })} suggestions={topics.map((t) => t.name)} placeholder="Routing topics this answers" /></div>
        {depts.length > 0 && (
          <div className="field span-all"><span className="field-label">Departments</span>
            <ChipSelect options={depts.map((d) => ({ value: String(d.id), label: d.name }))} value={(m.department_ids ?? []).map(String)}
              onChange={(v) => set({ department_ids: v.map(Number) })} />
            <span className="hint">Empty = the whole organization.</span></div>
        )}
      </div>
      <div className="form-section">
        <h3>Who can read it</h3>
        <Segmented label="Visibility" value={m.visibility} onChange={(v) => set({ visibility: v })}
          options={[{ value: 'all', label: 'Every employee' }, { value: 'restricted', label: 'Restricted' }]} />
        {m.visibility === 'restricted' && (
          <div className="field"><span className="field-label">Allowed roles</span>
            <ChipSelect options={roleOpts} value={m.allowed_roles ?? []} onChange={(v) => set({ allowed_roles: v })} />
            <span className="hint">People with the kb.restricted permission can always read it.</span></div>
        )}
        <Toggle checked={m.ai_enabled} onChange={(v) => set({ ai_enabled: v })} label="The AI may use this source (when active and not expired)" />
      </div>
    </>
  );
}
