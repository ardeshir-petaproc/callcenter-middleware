import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { Pencil, PhoneForwarded, PhoneOutgoing, Plus, Trash2 } from 'lucide-react';
import type { Callback, CallbackStatus } from '../../api/types';
import { del, get, post } from '../../api/client';
import { useApi } from '../../lib/hooks';
import { onLive } from '../../state/ws';
import { useQueueNames, useUserNames } from '../../state/lookups';
import { useMe } from '../../state/auth';
import { isSupPlus } from '../../lib/roles';
import { fmtDateTime, fmtRelative, humanize } from '../../lib/format';
import { Card, Empty, ErrorBox, PageHeader, Segmented, SkeletonRows } from '../../components/ui/primitives';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';
import { CallbackDialog } from './CallbackDialog';

const tone: Record<CallbackStatus, string> = { pending: 'warn', in_progress: 'info', done: 'ok', failed: 'danger', cancelled: '' };
type Filter = CallbackStatus | 'all';

export default function CallbacksPage() {
  const me = useMe();
  const [status, setStatus] = useState<Filter>('pending');
  const { data, loading, error, reload, setData } = useApi((s) => get<Callback[]>('/callbacks', { status: status === 'all' ? undefined : status }, s), [status]);
  const [edit, setEdit] = useState<Callback | null | 'new'>(null);
  const queueName = useQueueNames();
  const userName = useUserNames();
  useEffect(() => onLive('callback.created', () => reload()), [reload]);

  const items = [...(data ?? [])].sort((a, b) => Date.parse(a.due_at ?? a.created_at) - Date.parse(b.due_at ?? b.created_at));
  const dial = async (c: Callback) => {
    try { await post(`/callbacks/${c.id}/dial`); toast.success(`Dialing ${c.name || c.number} — answer your phone`); reload(); } catch (e) { toast.error(e); }
  };
  const remove = async (c: Callback) => {
    if (!(await confirm({ title: 'Delete callback?', message: `Delete the callback request from ${c.name || c.number}?`, danger: true, confirmLabel: 'Delete' }))) return;
    try { await del(`/callbacks/${c.id}`); setData((l) => l?.filter((x) => x.id !== c.id)); toast.success('Callback deleted'); } catch (e) { toast.error(e); }
  };

  return (
    <>
      <PageHeader title="Callbacks" sub="Callers who asked to be called back" actions={<button className="btn primary" onClick={() => setEdit('new')}><Plus />New callback</button>} />
      <Card flush title={
        <Segmented<Filter> label="Status" value={status} onChange={setStatus}
          options={(['pending', 'in_progress', 'done', 'failed', 'cancelled', 'all'] as Filter[]).map((s) => ({ value: s, label: humanize(s) }))} />
      }>
        {error ? <div style={{ padding: 16 }}><ErrorBox error={error} onRetry={reload} /></div>
          : loading && !data ? <SkeletonRows rows={6} />
          : items.length === 0 ? <Empty title="No callbacks" icon={<PhoneForwarded />}>Nothing in this list right now.</Empty>
          : (
            <div className="table-wrap">
              <table className="table">
                <thead><tr><th>Caller</th><th>Status</th><th className="hide-sm">Queue</th><th>Due</th><th className="hide-sm">Assigned</th><th className="num hide-sm">Attempts</th><th className="hide-sm">Notes</th><th /></tr></thead>
                <tbody>
                  {items.map((c) => {
                    const overdue = c.status === 'pending' && c.due_at && Date.parse(c.due_at) < Date.now();
                    return (
                      <tr key={c.id}>
                        <td><div className="strong">{c.name || c.number}</div>{c.name && <div className="sub">{c.number}</div>}</td>
                        <td><span className={`badge ${tone[c.status] ?? ''}`}>{humanize(c.status)}</span></td>
                        <td className="hide-sm">{queueName(c.queue_id)}</td>
                        <td title={fmtDateTime(c.due_at ?? c.created_at)} style={overdue ? { color: 'var(--danger)', fontWeight: 600 } : undefined}>
                          {c.due_at ? fmtRelative(c.due_at) : <span className="muted">asap · {fmtRelative(c.created_at)}</span>}
                        </td>
                        <td className="hide-sm">{c.assigned_to ? userName(c.assigned_to) : <span className="muted">—</span>}</td>
                        <td className="num hide-sm">{c.attempts}</td>
                        <td className="hide-sm" style={{ maxWidth: 240 }}>
                          <span className="ellipsis" style={{ display: 'block' }}>{c.notes || (c.call_id ? <Link to={`/calls/${encodeURIComponent(c.call_id)}`}>Original call</Link> : '—')}</span>
                        </td>
                        <td className="actions">
                          <div className="btn-group" style={{ flexWrap: 'nowrap' }}>
                            {(c.status === 'pending' || c.status === 'failed') && <button className="btn sm success" onClick={() => dial(c)}><PhoneOutgoing />Dial</button>}
                            <button className="btn sm icon" aria-label="Edit" onClick={() => setEdit(c)}><Pencil /></button>
                            {isSupPlus(me.role) && <button className="btn sm icon" aria-label="Delete" onClick={() => remove(c)}><Trash2 /></button>}
                          </div>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
      </Card>
      <CallbackDialog open={edit !== null} initial={edit === 'new' ? null : edit} onClose={() => setEdit(null)} onSaved={() => reload()} />
    </>
  );
}
