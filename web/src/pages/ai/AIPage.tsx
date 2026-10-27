import { useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { AlertTriangle, Bot, Mic } from 'lucide-react';
import type { AIProfile, AIStatus } from '../../api/types';
import { get } from '../../api/client';
import { useCrud } from '../../lib/crud';
import { useApi, useDraft } from '../../lib/hooks';
import { useMe } from '../../state/auth';
import { isAdmin } from '../../lib/roles';
import { useStore } from '../../lib/store';
import { live } from '../../state/live';
import { ConfigLayout, EditorFooter } from '../../components/ConfigLayout';
import { Card, Empty } from '../../components/ui/primitives';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';
import { AIProfileForm, blankProfile } from './AIProfileForm';
import { AITestDialog } from './AITestDialog';

export default function AIPage() {
  const { id } = useParams();
  const nav = useNavigate();
  const me = useMe();
  const canEdit = isAdmin(me.role);
  const crud = useCrud<AIProfile>('/ai/profiles', 'aiProfiles');
  const status = useApi((s) => get<AIStatus>('/ai/status', undefined, s), []);
  const sessions = useStore(live, (s) => s.aiSessions);
  const isNew = id === 'new';
  const source = useMemo(() => (isNew ? { ...blankProfile } : crud.items.find((x) => String(x.id) === id)), [isNew, id, crud.items]);
  const { draft, set, dirty } = useDraft<AIProfile>(source, id ?? '');
  const [busy, setBusy] = useState(false);
  const [testing, setTesting] = useState(false);

  const save = async () => {
    if (!draft) return;
    if (!draft.name.trim()) { toast.error('Name is required'); return; }
    setBusy(true);
    try {
      const saved = await crud.save(isNew ? { ...draft, id: undefined } as Partial<AIProfile> : draft);
      toast.success(isNew ? 'AI agent created' : 'AI agent saved');
      if (isNew) nav(`/ai/${saved.id}`, { replace: true });
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  const remove = async () => {
    if (!draft || !(await confirm({ title: 'Delete AI agent?', message: `Delete "${draft.name}"? Flows and queues pointing to it will need a new target.`, danger: true, confirmLabel: 'Delete' }))) return;
    try { await crud.remove(draft.id); toast.success('AI agent deleted'); nav('/ai', { replace: true }); } catch (e) { toast.error(e); }
  };
  const aiOff = status.data && !status.data.enabled;

  return (
    <ConfigLayout
      title="AI agents" sub={`Voice agents powered by the OpenAI Realtime API · ${sessions} live session${sessions === 1 ? '' : 's'}`} basePath="/ai"
      items={crud.items} loading={crud.loading} error={crud.error} onRetry={crud.reload}
      selected={id} canCreate={canEdit} newLabel="New AI agent" emptyIcon={<Bot />}
      search={(p) => p.name}
      renderItem={(p) => (
        <div className="grow">
          <div className="strong ellipsis">{p.name}</div>
          <div className="small muted">{p.voice || 'default voice'} · {p.language || '—'}{p.enabled ? '' : ' · disabled'}</div>
        </div>
      )}
    >
      {aiOff && (
        <div className="alert warn"><AlertTriangle /><div><strong>AI is disabled on the server.</strong> Set <code>OPENAI_API_KEY</code> to enable voice agents — calls routed to AI fall back to a queue until then.</div></div>
      )}
      {!id ? <div className="card"><Empty title="Select an AI agent" icon={<Bot />}>Pick a profile to edit or test it.</Empty></div>
        : !draft ? <div className="card"><Empty title={crud.loading ? 'Loading…' : 'Profile not found'} /></div>
        : (
          <Card title={isNew ? 'New AI agent' : draft.name || 'AI agent'} flush actions={!isNew && (
            <button className="btn" disabled={!!aiOff || dirty} title={aiOff ? 'AI is disabled on the server' : dirty ? 'Save changes first' : undefined} onClick={() => setTesting(true)}>
              <Mic />Test in browser
            </button>
          )}>
            <div className="card-body"><AIProfileForm p={draft} set={set} disabled={!canEdit} status={status.data} /></div>
            <EditorFooter onSave={save} onDelete={remove} busy={busy} canEdit={canEdit} dirty={dirty} isNew={isNew} />
          </Card>
        )}
      {testing && draft && <AITestDialog profile={draft} onClose={() => setTesting(false)} />}
    </ConfigLayout>
  );
}
