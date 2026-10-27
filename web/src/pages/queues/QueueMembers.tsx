import { useEffect, useMemo, useState } from 'react';
import { Plus, Trash2, Users } from 'lucide-react';
import type { QueueMember } from '../../api/types';
import { get, put } from '../../api/client';
import { useApi } from '../../lib/hooks';
import { useLookup } from '../../state/lookups';
import { useStore } from '../../lib/store';
import { live } from '../../state/live';
import { Card, Empty, SkeletonRows } from '../../components/ui/primitives';
import { AgentStatusBadge } from '../../components/ui/Badges';
import { toast } from '../../components/ui/Toast';

type Row = { user_id: number; penalty: number };

export function QueueMembers({ queueId, canEdit }: { queueId: number; canEdit: boolean }) {
  const users = useLookup('users');
  const agents = useStore(live, (s) => s.agents);
  const { data, loading, setData } = useApi((s) => get<QueueMember[]>('/members', undefined, s), [queueId]);
  const original = useMemo<Row[]>(() => (data ?? []).filter((m) => m.queue_id === queueId).map(({ user_id, penalty }) => ({ user_id, penalty })), [data, queueId]);
  const [rows, setRows] = useState<Row[]>(original);
  const [pick, setPick] = useState('');
  const [busy, setBusy] = useState(false);
  useEffect(() => setRows(original), [original]);

  const dirty = JSON.stringify(rows) !== JSON.stringify(original);
  const available = users.filter((u) => !rows.some((r) => r.user_id === u.id));
  const nameOf = (id: number) => { const u = users.find((x) => x.id === id); return u ? u.display_name || u.username : `User #${id}`; };

  const save = async () => {
    setBusy(true);
    try {
      const res = await put<QueueMember[]>(`/queues/${queueId}/members`, rows);
      const others = (data ?? []).filter((m) => m.queue_id !== queueId);
      setData([...others, ...(Array.isArray(res) ? res : rows.map((r) => ({ ...r, queue_id: queueId })))]);
      toast.success('Members updated');
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };

  return (
    <Card title={`Members (${rows.length})`} icon={<Users />} flush actions={canEdit && dirty && (
      <button className="btn primary sm" disabled={busy} onClick={save}>Save members</button>
    )}>
      {loading && !data ? <SkeletonRows rows={3} /> : (
        <>
          {rows.length === 0 ? <Empty title="No members" icon={<Users />}>Add agents who should take calls from this queue.</Empty> : (
            <div className="table-wrap">
              <table className="table">
                <thead><tr><th>Agent</th><th>Status</th><th style={{ width: 120 }}>Penalty</th>{canEdit && <th />}</tr></thead>
                <tbody>
                  {rows.map((r) => (
                    <tr key={r.user_id}>
                      <td className="strong">{nameOf(r.user_id)}</td>
                      <td>{agents[r.user_id] ? <AgentStatusBadge status={agents[r.user_id].status} /> : <span className="muted">—</span>}</td>
                      <td>
                        <input className="input sm" type="number" min={0} value={r.penalty} disabled={!canEdit} aria-label="Penalty"
                          onChange={(e) => setRows(rows.map((x) => (x.user_id === r.user_id ? { ...x, penalty: Number(e.target.value) } : x)))} />
                      </td>
                      {canEdit && (
                        <td className="actions">
                          <button className="btn ghost icon sm" aria-label="Remove member" onClick={() => setRows(rows.filter((x) => x.user_id !== r.user_id))}><Trash2 /></button>
                        </td>
                      )}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
          {canEdit && (
            <div className="row" style={{ padding: 12, borderTop: '1px solid var(--border)' }}>
              <select className="select sm" value={pick} onChange={(e) => setPick(e.target.value)} aria-label="Add member">
                <option value="">Add an agent…</option>
                {available.map((u) => <option key={u.id} value={u.id}>{u.display_name || u.username}{u.extension ? ` (${u.extension})` : ''}</option>)}
              </select>
              <button className="btn sm" disabled={!pick} onClick={() => { setRows([...rows, { user_id: Number(pick), penalty: 0 }]); setPick(''); }}><Plus />Add</button>
            </div>
          )}
        </>
      )}
    </Card>
  );
}
