import { useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { GitBranch, Workflow } from 'lucide-react';
import type { IVRFlow } from '../../api/types';
import { useCrud } from '../../lib/crud';
import { useDraft } from '../../lib/hooks';
import { ConfigLayout, EditorFooter } from '../../components/ConfigLayout';
import { Card, Empty } from '../../components/ui/primitives';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';
import { blankFlow, FlowForm } from './FlowForm';
import { FlowPreview } from './FlowPreview';

export default function FlowsPage() {
  const { id } = useParams();
  const nav = useNavigate();
  const crud = useCrud<IVRFlow>('/flows', 'flows');
  const isNew = id === 'new';
  const source = useMemo(() => {
    if (isNew) return { ...blankFlow };
    const f = crud.items.find((x) => String(x.id) === id);
    return f ? { ...f, options: f.options ?? {} } : undefined;
  }, [isNew, id, crud.items]);
  const { draft, set, dirty } = useDraft<IVRFlow>(source, id ?? '');
  const [busy, setBusy] = useState(false);

  const save = async () => {
    if (!draft) return;
    if (!draft.name.trim()) { toast.error('Name is required'); return; }
    setBusy(true);
    try {
      const saved = await crud.save(isNew ? { ...draft, id: undefined } as Partial<IVRFlow> : draft);
      toast.success(isNew ? 'Flow created' : 'Flow saved');
      if (isNew) nav(`/flows/${saved.id}`, { replace: true });
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  const remove = async () => {
    if (!draft || !(await confirm({ title: 'Delete IVR flow?', message: `Delete "${draft.name}"? Calls to DNIS ${draft.dnis || '(none)'} will no longer be handled by it.`, danger: true, confirmLabel: 'Delete' }))) return;
    try { await crud.remove(draft.id); toast.success('Flow deleted'); nav('/flows', { replace: true }); } catch (e) { toast.error(e); }
  };

  return (
    <ConfigLayout
      title="IVR flows" sub="Greetings, menus and business-hours routing" basePath="/flows"
      items={crud.items} loading={crud.loading} error={crud.error} onRetry={crud.reload}
      selected={id} canCreate newLabel="New flow" emptyIcon={<Workflow />}
      search={(f) => `${f.name} ${f.dnis}`}
      renderItem={(f) => (
        <div className="grow">
          <div className="strong ellipsis">{f.name}</div>
          <div className="small muted">DNIS {f.dnis || '—'} · {Object.keys(f.options ?? {}).length} keys{f.enabled ? '' : ' · disabled'}</div>
        </div>
      )}
    >
      {!id ? <div className="card"><Empty title="Select a flow" icon={<Workflow />}>Choose an IVR flow or create a new one.</Empty></div>
        : !draft ? <div className="card"><Empty title={crud.loading ? 'Loading…' : 'Flow not found'} /></div>
        : (
          <>
            <Card title="Caller journey" icon={<GitBranch />}><FlowPreview flow={draft} /></Card>
            <Card title={isNew ? 'New flow' : draft.name || 'Flow'} flush>
              <div className="card-body"><FlowForm f={draft} set={set} disabled={false} /></div>
              <EditorFooter onSave={save} onDelete={remove} busy={busy} canEdit dirty={dirty} isNew={isNew} />
            </Card>
          </>
        )}
    </ConfigLayout>
  );
}
