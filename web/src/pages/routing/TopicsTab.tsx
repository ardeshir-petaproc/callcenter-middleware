import { useState } from 'react';
import { Pencil, Plus, Tags, Trash2 } from 'lucide-react';
import type { Topic } from '../../api/types_routing';
import { useCrud } from '../../lib/crud';
import { PERMS } from '../../lib/perms';
import { useCan } from '../../state/auth';
import { useDepartmentNames, useLookup } from '../../state/lookups';
import { Card, Empty, ErrorBox, Field, SkeletonRows } from '../../components/ui/primitives';
import { Dialog } from '../../components/ui/Dialog';
import { TagInput } from '../../components/ui/TagInput';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';
import { RevisionsButton } from '../../components/RevisionsButton';

const blank: Omit<Topic, 'id'> & { id?: number } = { name: '', description: '', keywords: [], department_id: 0 };

/** Kinds of requests callers make, with words that signal them. */
export function TopicsTab() {
  const canManage = useCan()(PERMS.routingManage);
  const crud = useCrud<Topic>('/routing/topics', 'topics');
  const depts = useLookup('departments');
  const deptName = useDepartmentNames();
  const [edit, setEdit] = useState<typeof blank | null>(null);
  const [busy, setBusy] = useState(false);
  const save = async () => {
    if (!edit) return;
    setBusy(true);
    try { await crud.save(edit as Partial<Topic>); toast.success('Topic saved'); setEdit(null); } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  const remove = async (t: Topic) => {
    if (!(await confirm({ title: 'Delete topic?', message: `Rules that reference “${t.name}” will no longer match it.`, danger: true, confirmLabel: 'Delete' }))) return;
    try { await crud.remove(t.id); } catch (e) { toast.error(e); }
  };
  return (
    <Card title="Topics" icon={<Tags />} flush actions={canManage && <button className="btn sm primary" onClick={() => setEdit({ ...blank })}><Plus />New topic</button>}>
      {crud.error ? <div style={{ padding: 16 }}><ErrorBox error={crud.error} onRetry={crud.reload} /></div>
        : crud.loading && !crud.data ? <SkeletonRows rows={4} />
        : crud.items.length === 0 ? <Empty title="No topics" icon={<Tags />}>Topics such as “Refund” or “Technical support” are detected from what callers say.</Empty>
        : (
          <div className="list">
            {crud.items.map((t) => (
              <div key={t.id} className="list-item" style={{ alignItems: 'flex-start' }}>
                <div className="grow">
                  <div className="row wrap"><span className="strong" dir="auto">{t.name}</span><span className="small muted">default owner: {deptName(t.department_id)}</span></div>
                  {t.description && <div className="small muted" dir="auto">{t.description}</div>}
                  <div className="chips" style={{ marginTop: 4 }}>{(t.keywords ?? []).map((k) => <span key={k} className="chip" dir="auto">{k}</span>)}</div>
                </div>
                <RevisionsButton entity="topic" id={t.id} label="" />
                {canManage && <button className="btn sm ghost icon" aria-label="Edit" onClick={() => setEdit({ ...t, keywords: t.keywords ?? [] })}><Pencil /></button>}
                {canManage && <button className="btn sm ghost icon" aria-label="Delete" onClick={() => remove(t)}><Trash2 /></button>}
              </div>
            ))}
          </div>
        )}
      <Dialog open={!!edit} onClose={() => setEdit(null)} title={edit?.id ? 'Edit topic' : 'New topic'}
        footer={<><button className="btn" onClick={() => setEdit(null)}>Cancel</button><button className="btn primary" disabled={busy || !edit?.name.trim()} onClick={save}>Save</button></>}>
        {edit && (
          <>
            <Field label="Name"><input className="input" dir="auto" value={edit.name} onChange={(e) => setEdit({ ...edit, name: e.target.value })} data-autofocus /></Field>
            <Field label="Description"><input className="input" dir="auto" value={edit.description} onChange={(e) => setEdit({ ...edit, description: e.target.value })} /></Field>
            <div className="field"><span className="field-label">Keywords</span>
              <TagInput value={edit.keywords ?? []} onChange={(v) => setEdit({ ...edit, keywords: v })} placeholder="Words or phrases, any language" /></div>
            <Field label="Default department" hint="Owner when no rule matches">
              <select className="select" value={edit.department_id || ''} onChange={(e) => setEdit({ ...edit, department_id: Number(e.target.value) || 0 })}>
                <option value="">None</option>
                {depts.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
              </select>
            </Field>
          </>
        )}
      </Dialog>
    </Card>
  );
}
