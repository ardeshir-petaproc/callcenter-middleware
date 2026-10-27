import { useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { Megaphone, Pause, Play } from 'lucide-react';
import type { Campaign, CampaignStats } from '../../api/types';
import { get, post } from '../../api/client';
import { useCrud } from '../../lib/crud';
import { useDraft } from '../../lib/hooks';
import { onLive } from '../../state/ws';
import { useQueueNames } from '../../state/lookups';
import { ConfigLayout, EditorFooter } from '../../components/ConfigLayout';
import { Card, Empty, Kpi } from '../../components/ui/primitives';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';
import { fmtCsat, fmtDuration, fmtNum } from '../../lib/format';
import { CampaignForm, blankCampaign } from './CampaignForm';
import { CampaignLeads } from './CampaignLeads';
import { CampaignProgress, CampaignStatusBadge } from './campaignUi';

/** Stats for every campaign, refreshed periodically and when calls end. */
function useAllStats(ids: number[]): [Record<number, CampaignStats>, () => void] {
  const [stats, setStats] = useState<Record<number, CampaignStats>>({});
  const [nonce, setNonce] = useState(0);
  const key = ids.join(',');
  useEffect(() => {
    let alive = true;
    const load = () => Promise.all(ids.map((id) => get<CampaignStats>(`/campaigns/${id}/stats`).then((s) => [id, s] as const).catch(() => null)))
      .then((res) => { if (alive) setStats(Object.fromEntries(res.filter(Boolean) as [number, CampaignStats][])); });
    void load();
    const t = window.setInterval(load, 15000);
    const off = onLive('call.ended', () => void load());
    return () => { alive = false; clearInterval(t); off(); };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [key, nonce]);
  return [stats, () => setNonce((n) => n + 1)];
}

export default function CampaignsPage() {
  const { id } = useParams();
  const nav = useNavigate();
  const crud = useCrud<Campaign>('/campaigns', 'campaigns');
  const queueName = useQueueNames();
  const [stats, reloadStats] = useAllStats(crud.items.map((c) => c.id));
  const isNew = id === 'new';
  const source = useMemo(() => (isNew ? { ...blankCampaign } : crud.items.find((x) => String(x.id) === id)), [isNew, id, crud.items]);
  const { draft, set, dirty } = useDraft<Campaign>(source, id ?? '');
  const [busy, setBusy] = useState(false);

  const save = async () => {
    if (!draft) return;
    if (!draft.name.trim() || !draft.queue_id) { toast.error('Name and queue are required'); return; }
    setBusy(true);
    try {
      const saved = await crud.save(isNew ? { ...draft, id: undefined } as Partial<Campaign> : draft);
      toast.success(isNew ? 'Campaign created' : 'Campaign saved');
      if (isNew) nav(`/campaigns/${saved.id}`, { replace: true });
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  const remove = async () => {
    if (!draft || !(await confirm({ title: 'Delete campaign?', message: `Delete "${draft.name}" and all of its leads?`, danger: true, confirmLabel: 'Delete' }))) return;
    try { await crud.remove(draft.id); toast.success('Campaign deleted'); nav('/campaigns', { replace: true }); } catch (e) { toast.error(e); }
  };
  const runAction = async (action: 'start' | 'pause') => {
    if (!draft) return;
    setBusy(true);
    try {
      const c = await post<Campaign>(`/campaigns/${draft.id}/${action}`);
      crud.setData((list) => list?.map((x) => (x.id === draft.id ? (c && 'id' in c ? c : { ...x, status: action === 'start' ? 'running' : 'paused' }) : x)));
      toast.success(action === 'start' ? 'Campaign started — dialing as agents become available' : 'Campaign paused');
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  const st = draft ? stats[draft.id] : undefined;
  const calls = st?.calls;

  return (
    <ConfigLayout
      title="Campaigns" sub="Progressive outbound dialing: agents are rung first, then connected to the next lead" basePath="/campaigns"
      items={crud.items} loading={crud.loading} error={crud.error} onRetry={crud.reload}
      selected={id} canCreate newLabel="New campaign" emptyIcon={<Megaphone />}
      search={(c) => `${c.name} ${c.description}`}
      renderItem={(c) => (
        <div className="grow col" style={{ gap: 4, minWidth: 0 }}>
          <div className="row between"><span className="strong ellipsis">{c.name}</span><CampaignStatusBadge status={c.status} /></div>
          <div className="small muted ellipsis">{queueName(c.queue_id)} · {stats[c.id]?.leads.total ?? 0} leads</div>
          <CampaignProgress stats={stats[c.id]} />
        </div>
      )}
    >
      {!id ? <div className="card"><Empty title="Select a campaign" icon={<Megaphone />}>Pick a campaign or create a new one.</Empty></div>
        : !draft ? <div className="card"><Empty title={crud.loading ? 'Loading…' : 'Campaign not found'} /></div>
        : (
          <>
            {!isNew && (
              <Card title={<span className="row">Progress <CampaignStatusBadge status={draft.status} /></span>} actions={
                draft.status === 'running'
                  ? <button className="btn" disabled={busy} onClick={() => runAction('pause')}><Pause />Pause</button>
                  : <button className="btn success" disabled={busy || dirty || !(st?.leads.pending)} title={dirty ? 'Save changes first' : !st?.leads.pending ? 'No pending leads' : undefined} onClick={() => runAction('start')}><Play />{draft.status === 'paused' ? 'Resume' : 'Start'}</button>
              }>
                <div className="col gap-lg">
                  <CampaignProgress stats={st} showLabel />
                  <div className="kpis" style={{ gridTemplateColumns: 'repeat(auto-fill, minmax(118px, 1fr))' }}>
                    <Kpi label="Pending" value={fmtNum(st?.leads.pending ?? 0)} />
                    <Kpi label="Dialing" value={fmtNum(st?.leads.dialing ?? 0)} />
                    <Kpi label="Answered" value={fmtNum((st?.leads.answered ?? 0) + (st?.leads.done ?? 0))} tone="ok" />
                    <Kpi label="Failed" value={fmtNum(st?.leads.failed ?? 0)} tone={st?.leads.failed ? 'danger' : undefined} />
                    <Kpi label="Calls" value={fmtNum(calls?.total ?? 0)} sub={calls ? `Avg talk ${fmtDuration(calls.avg_talk_sec)}` : undefined} />
                    <Kpi label="CSAT" value={fmtCsat(calls?.csat_avg, calls?.csat_count)} sub={calls?.csat_count ? `${calls.csat_count} ratings` : undefined} />
                  </div>
                </div>
              </Card>
            )}
            <Card title={isNew ? 'New campaign' : 'Settings'} flush>
              <div className="card-body"><CampaignForm c={draft} set={set} disabled={false} /></div>
              <EditorFooter onSave={save} onDelete={remove} busy={busy} canEdit dirty={dirty} isNew={isNew} />
            </Card>
            {!isNew && <CampaignLeads campaignId={draft.id} canEdit counts={st?.leads ?? {}} onChanged={reloadStats} />}
          </>
        )}
    </ConfigLayout>
  );
}
