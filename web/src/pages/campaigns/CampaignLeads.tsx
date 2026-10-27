import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Trash2, Upload, UserRoundSearch } from 'lucide-react';
import type { Lead, LeadStatus } from '../../api/types';
import { del, get } from '../../api/client';
import { useApi } from '../../lib/hooks';
import { fmtDateTime, fmtRelative, humanize } from '../../lib/format';
import { Card, Empty, ErrorBox, Pager, Segmented, SkeletonRows } from '../../components/ui/primitives';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';
import { LeadStatusBadge } from './campaignUi';
import { LeadImportDialog } from './LeadImportDialog';

type Filter = LeadStatus | 'all';
const FILTERS: Filter[] = ['all', 'pending', 'dialing', 'answered', 'failed', 'done'];
const LIMIT = 50;

export function CampaignLeads({ campaignId, canEdit, counts, onChanged }: {
  campaignId: number; canEdit: boolean; counts: Partial<Record<string, number>>; onChanged: () => void;
}) {
  const [status, setStatus] = useState<Filter>('all');
  const [offset, setOffset] = useState(0);
  const [importing, setImporting] = useState(false);
  const { data, loading, error, reload } = useApi(
    (s) => get<Lead[]>(`/campaigns/${campaignId}/leads`, { status: status === 'all' ? undefined : status, limit: LIMIT, offset }, s),
    [campaignId, status, offset],
  );
  const total = (status === 'all' ? counts.total : counts[status]) ?? (data?.length ?? 0);
  const rows = data ?? [];

  const clear = async () => {
    const what = status === 'all' ? 'ALL leads' : `all ${status} leads`;
    if (!(await confirm({ title: 'Remove leads?', message: `Remove ${what} from this campaign?`, danger: true, confirmLabel: 'Remove' }))) return;
    try {
      await del(`/campaigns/${campaignId}/leads${status === 'all' ? '' : `?status=${status}`}`);
      toast.success('Leads removed');
      reload(); onChanged();
    } catch (e) { toast.error(e); }
  };

  return (
    <Card flush title={`Leads (${counts.total ?? 0})`} icon={<UserRoundSearch />} actions={canEdit && (
      <>
        <button className="btn sm" onClick={clear} disabled={!rows.length}><Trash2 />Remove {status === 'all' ? 'all' : status}</button>
        <button className="btn sm primary" onClick={() => setImporting(true)}><Upload />Import CSV</button>
      </>
    )}>
      <div style={{ padding: '10px 16px', borderBottom: '1px solid var(--border)' }}>
        <Segmented<Filter> label="Lead status" value={status} onChange={(v) => { setStatus(v); setOffset(0); }}
          options={FILTERS.map((f) => ({ value: f, label: <>{humanize(f)} <span className="muted num">{(f === 'all' ? counts.total : counts[f]) ?? 0}</span></> }))} />
      </div>
      {error ? <div style={{ padding: 16 }}><ErrorBox error={error} onRetry={reload} /></div>
        : loading && !data ? <SkeletonRows rows={4} />
        : rows.length === 0 ? <Empty title="No leads" icon={<UserRoundSearch />}>{canEdit ? 'Import a CSV of numbers to call.' : undefined}</Empty>
        : (
          <div className="table-wrap">
            <table className="table">
              <thead><tr><th>Number</th><th>Name</th><th>Status</th><th className="num">Attempts</th><th className="hide-sm">Next attempt</th><th className="hide-sm">Notes</th><th className="hide-sm">Updated</th></tr></thead>
              <tbody>
                {rows.map((l) => (
                  <tr key={l.id}>
                    <td className="tnum">{l.call_id ? <Link to={`/calls/${encodeURIComponent(l.call_id)}`}>{l.number}</Link> : l.number}</td>
                    <td>{l.name || <span className="muted">—</span>}</td>
                    <td><LeadStatusBadge status={l.status} /></td>
                    <td className="num">{l.attempts}</td>
                    <td className="hide-sm" title={fmtDateTime(l.next_attempt)}>{l.status === 'pending' && l.attempts > 0 ? fmtRelative(l.next_attempt) : '—'}</td>
                    <td className="hide-sm small" style={{ maxWidth: 260 }}>{l.notes || '—'}</td>
                    <td className="hide-sm small muted">{fmtRelative(l.updated_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      {total > LIMIT && <Pager offset={offset} limit={LIMIT} total={total} onChange={setOffset} />}
      <LeadImportDialog campaignId={campaignId} open={importing} onClose={() => setImporting(false)} onImported={() => { reload(); onChanged(); }} />
    </Card>
  );
}
