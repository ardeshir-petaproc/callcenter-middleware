import { useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { ListTree } from 'lucide-react';
import type { Queue } from '../../api/types';
import { useCrud } from '../../lib/crud';
import { useDraft } from '../../lib/hooks';
import { useMe } from '../../state/auth';
import { isAdmin } from '../../lib/roles';
import { useStore } from '../../lib/store';
import { live } from '../../state/live';
import { ConfigLayout, EditorFooter } from '../../components/ConfigLayout';
import { Card, Empty } from '../../components/ui/primitives';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';
import { blankQueue, QueueForm } from './QueueForm';
import { QueueMembers } from './QueueMembers';

export default function QueuesPage() {
  const { id } = useParams();
  const nav = useNavigate();
  const me = useMe();
  const canEdit = isAdmin(me.role);
  const crud = useCrud<Queue>('/queues', 'queues');
  const views = useStore(live, (s) => s.queues);
  const isNew = id === 'new';
  const source = useMemo(() => (isNew ? { ...blankQueue } as Queue : crud.items.find((q) => String(q.id) === id)), [isNew, id, crud.items]);
  const { draft, set, dirty } = useDraft<Queue>(source, id ?? '');
  const [busy, setBusy] = useState(false);

  const save = async () => {
    if (!draft) return;
    if (!draft.name.trim()) { toast.error('Name is required'); return; }
    setBusy(true);
    try {
      const saved = await crud.save(isNew ? { ...draft, id: undefined } as Partial<Queue> : draft);
      toast.success(isNew ? 'Queue created' : 'Queue saved');
      if (isNew) nav(`/queues/${saved.id}`, { replace: true });
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  const remove = async () => {
    if (!draft || !(await confirm({ title: 'Delete queue?', message: `Delete "${draft.name}"? Calls routed to it will fail over.`, danger: true, confirmLabel: 'Delete' }))) return;
    try { await crud.remove(draft.id); toast.success('Queue deleted'); nav('/queues', { replace: true }); } catch (e) { toast.error(e); }
  };

  return (
    <ConfigLayout
      title="Queues" sub="Routing groups, service levels and membership" basePath="/queues"
      items={crud.items} loading={crud.loading} error={crud.error} onRetry={crud.reload}
      selected={id} canCreate={canEdit} newLabel="New queue" emptyIcon={<ListTree />}
      search={(q) => q.name}
      renderItem={(q) => {
        const v = views.find((x) => x.id === q.id);
        return (
          <>
            <div className="grow">
              <div className="strong ellipsis">{q.name}</div>
              <div className="small muted">{q.strategy.replace('_', ' ')} · SLA {q.sla_sec}s{q.enabled ? '' : ' · disabled'}</div>
            </div>
            {v && <span className={`badge ${v.waiting ? 'warn' : ''}`} title="Waiting">{v.waiting}</span>}
          </>
        );
      }}
    >
      {!id ? <div className="card"><Empty title="Select a queue" icon={<ListTree />}>Choose a queue on the left{canEdit ? ' or create a new one' : ''}.</Empty></div>
        : !draft ? <div className="card"><Empty title={crud.loading ? 'Loading…' : 'Queue not found'} /></div>
        : (
          <>
            <Card title={isNew ? 'New queue' : draft.name || 'Queue'} flush>
              <div className="card-body"><QueueForm q={draft} set={set} disabled={!canEdit} /></div>
              <EditorFooter onSave={save} onDelete={remove} busy={busy} canEdit={canEdit} dirty={dirty} isNew={isNew} />
            </Card>
            {!isNew && <QueueMembers queueId={draft.id} canEdit={canEdit} />}
          </>
        )}
    </ConfigLayout>
  );
}
