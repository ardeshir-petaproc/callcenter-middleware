import { useEffect } from 'react';
import { Link } from 'react-router-dom';
import { Check, PhoneForwarded, PhoneOutgoing } from 'lucide-react';
import type { Callback } from '../../api/types';
import { get, post, put } from '../../api/client';
import { useApi } from '../../lib/hooks';
import { useMe } from '../../state/auth';
import { onLive } from '../../state/ws';
import { useQueueNames } from '../../state/lookups';
import { Card, Empty, SkeletonRows } from '../../components/ui/primitives';
import { toast } from '../../components/ui/Toast';
import { fmtRelative } from '../../lib/format';

export function MyCallbacks() {
  const me = useMe();
  const queueName = useQueueNames();
  const { data, loading, reload, setData } = useApi((s) => get<Callback[]>('/callbacks', { status: 'pending' }, s), []);
  useEffect(() => onLive('callback.created', () => reload()), [reload]);

  const mine = (data ?? []).filter((c) => !c.assigned_to || c.assigned_to === me.id)
    .sort((a, b) => Date.parse(a.due_at ?? a.created_at) - Date.parse(b.due_at ?? b.created_at));

  const dial = async (c: Callback) => {
    try {
      await post(`/callbacks/${c.id}/dial`);
      toast.success(`Dialing ${c.name || c.number} — answer your phone`);
      setData((list) => list?.filter((x) => x.id !== c.id));
    } catch (e) { toast.error(e); }
  };
  const done = async (c: Callback) => {
    try {
      await put(`/callbacks/${c.id}`, { ...c, status: 'done' });
      setData((list) => list?.filter((x) => x.id !== c.id));
      toast.success('Callback completed');
    } catch (e) { toast.error(e); }
  };

  return (
    <Card title={`My callbacks${mine.length ? ` (${mine.length})` : ''}`} icon={<PhoneForwarded />} flush actions={<Link className="btn sm" to="/callbacks">All</Link>}>
      {loading && !data ? <SkeletonRows rows={3} /> : mine.length === 0 ? (
        <Empty title="No pending callbacks" icon={<PhoneForwarded />} />
      ) : (
        <div className="list">
          {mine.slice(0, 8).map((c) => {
            const overdue = c.due_at && Date.parse(c.due_at) < Date.now();
            return (
              <div key={c.id} className="list-item">
                <div className="grow">
                  <div className="strong ellipsis">{c.name || c.number}</div>
                  <div className="small muted ellipsis">
                    {c.name ? `${c.number} · ` : ''}{c.queue_id ? queueName(c.queue_id) : 'No queue'}
                    {c.due_at ? <> · <span style={overdue ? { color: 'var(--danger)' } : undefined}>due {fmtRelative(c.due_at)}</span></> : ` · ${fmtRelative(c.created_at)}`}
                    {c.attempts ? ` · ${c.attempts} attempt${c.attempts > 1 ? 's' : ''}` : ''}
                  </div>
                </div>
                <button className="btn sm icon" title="Mark done" aria-label="Mark done" onClick={() => done(c)}><Check /></button>
                <button className="btn sm success" onClick={() => dial(c)}><PhoneOutgoing />Dial</button>
              </div>
            );
          })}
        </div>
      )}
    </Card>
  );
}
