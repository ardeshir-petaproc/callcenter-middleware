import { get } from '../../api/client';
import type { AIProfile } from '../../api/types';
import type { KBCategory } from '../../api/types_work';
import { useApi } from '../../lib/hooks';
import { useLookup } from '../../state/lookups';
import { ChipSelect, TagInput } from '../../components/ui/TagInput';

/** Which knowledge-base sources the agent's search_knowledge tool may use. */
export function KnowledgeScope({ p, set, disabled }: { p: AIProfile; set: (x: Partial<AIProfile>) => void; disabled: boolean }) {
  const depts = useLookup('departments');
  const cats = useApi((s) => get<KBCategory[]>('/kb/categories', undefined, s), []);
  const usesKB = (p.tools ?? []).includes('search_knowledge');
  return (
    <div className="form-section">
      <h3>Knowledge base scope</h3>
      <p className="small muted">
        The <code>search_knowledge</code> tool {usesKB ? 'is enabled and' : '(not enabled below)'} only finds active, unexpired, AI-enabled sources.
        Narrow it further by department and category; leave both empty for every AI-enabled source.
      </p>
      <div className="field"><span className="field-label">Departments</span>
        {depts.length ? <ChipSelect options={depts.map((d) => ({ value: String(d.id), label: d.name }))} disabled={disabled}
          value={(p.kb_department_ids ?? []).map(String)} onChange={(v) => set({ kb_department_ids: v.map(Number) })} />
          : <span className="small muted">No departments defined.</span>}
      </div>
      <div className="field"><span className="field-label">Categories</span>
        <TagInput value={p.kb_categories ?? []} onChange={(v) => set({ kb_categories: v })} disabled={disabled} placeholder="All categories"
          suggestions={(cats.data ?? []).map((c) => c.category).filter(Boolean)} />
      </div>
    </div>
  );
}
