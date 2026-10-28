import { useEffect, useState } from 'react';
import { AlarmClock, Save } from 'lucide-react';
import { TICKET_PRIORITIES, type SLAPolicy, type TicketPriority } from '../../api/types_work';
import { useCrud } from '../../lib/crud';
import { fmtLongDuration } from '../../lib/format';
import { Card, ErrorBox, SkeletonRows } from '../../components/ui/primitives';
import { PriorityBadge } from '../tickets/ticketUi';
import { toast } from '../../components/ui/Toast';

type Row = { id?: number; priority: TicketPriority; first_response_minutes: number; resolution_minutes: number };

/** Ticket SLA targets per priority (minutes; 0 = no target). */
export function SlaTab() {
  const crud = useCrud<SLAPolicy>('/sla-policies');
  const [rows, setRows] = useState<Row[]>([]);
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    setRows(TICKET_PRIORITIES.map((p) => crud.items.find((x) => x.priority === p) ?? { priority: p, first_response_minutes: 0, resolution_minutes: 0 }));
  }, [crud.items]);
  const dirty = rows.some((r) => {
    const cur = crud.items.find((x) => x.priority === r.priority);
    return !cur ? r.first_response_minutes > 0 || r.resolution_minutes > 0
      : cur.first_response_minutes !== r.first_response_minutes || cur.resolution_minutes !== r.resolution_minutes;
  });
  const set = (p: TicketPriority, patch: Partial<Row>) => setRows((rs) => rs.map((r) => (r.priority === p ? { ...r, ...patch } : r)));
  const save = async () => {
    setBusy(true);
    try {
      for (const r of rows) {
        const cur = crud.items.find((x) => x.priority === r.priority);
        if (cur && cur.first_response_minutes === r.first_response_minutes && cur.resolution_minutes === r.resolution_minutes) continue;
        if (!cur && !r.first_response_minutes && !r.resolution_minutes) continue;
        await crud.save({ ...r, id: cur?.id } as Partial<SLAPolicy>);
      }
      toast.success('SLA targets saved');
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };

  if (crud.error) return <ErrorBox error={crud.error} onRetry={crud.reload} />;
  return (
    <Card title="Ticket SLA policies" icon={<AlarmClock />} flush>
      {crud.loading && !crud.data ? <SkeletonRows rows={4} /> : (
        <div className="table-wrap">
          <table className="table">
            <thead><tr><th>Priority</th><th>First response (minutes)</th><th>Resolution (minutes)</th></tr></thead>
            <tbody>
              {rows.map((r) => (
                <tr key={r.priority}>
                  <td><PriorityBadge priority={r.priority} /></td>
                  {(['first_response_minutes', 'resolution_minutes'] as const).map((k) => (
                    <td key={k}>
                      <div className="row">
                        <input className="input sm" type="number" min={0} style={{ width: 110 }} value={r[k]} aria-label={`${r.priority} ${k}`}
                          onChange={(e) => set(r.priority, { [k]: Math.max(0, Number(e.target.value)) })} />
                        <span className="small muted">{r[k] ? fmtLongDuration(r[k] * 60) : 'no target'}</span>
                      </div>
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <div className="card-foot"><span className="small muted grow">New tickets get due dates from these targets.</span><button className="btn primary" disabled={!dirty || busy} onClick={save}><Save />Save</button></div>
    </Card>
  );
}
