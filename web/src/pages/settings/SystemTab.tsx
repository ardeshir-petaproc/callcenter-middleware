import { Fragment, useState } from 'react';
import { RefreshCw, Server, ScrollText } from 'lucide-react';
import type { AuditEntry, SystemInfo } from '../../api/types';
import { get } from '../../api/client';
import { useApi } from '../../lib/hooks';
import { fmtDateTime, fmtLongDuration, fmtNum } from '../../lib/format';
import { Card, Empty, ErrorBox, Pager, SkeletonRows } from '../../components/ui/primitives';

function Health({ ok, label }: { ok: boolean; label: string }) {
  return <span className={`badge ${ok ? 'ok' : 'danger'}`}><span className="dot" />{label}: {ok ? 'OK' : 'Down'}</span>;
}

export function SystemTab() {
  const { data: s, loading, error, reload } = useApi((sig) => get<SystemInfo>('/system/info', undefined, sig), []);
  if (error) return <ErrorBox error={error} onRetry={reload} />;
  if (loading && !s) return <div className="card"><SkeletonRows rows={5} /></div>;
  if (!s) return null;
  const rows: [string, string][] = [
    ['Version', s.version || '—'], ['Uptime', fmtLongDuration(s.uptime_sec)], ['Started', fmtDateTime(s.started_at)],
    ['WebSocket clients', fmtNum(s.ws_clients)], ['Default queue', s.default_queue || '—'], ['Queue context', s.queue_context || '—'],
    ['Outbound dial template', s.outbound_template || '—'],
  ];
  return (
    <Card title="System" icon={<Server />} actions={<button className="btn sm" onClick={reload}><RefreshCw />Refresh</button>}>
      <div className="col gap-lg">
        <div className="row wrap">
          <Health ok={s.ari_connected} label="Asterisk ARI" />
          <Health ok={s.db_ok} label="Database" />
          <span className={`badge ${s.ai_enabled ? 'ok' : 'warn'}`}><span className="dot" />AI: {s.ai_enabled ? 'Enabled' : 'Disabled'}</span>
        </div>
        <dl className="kv">{rows.map(([k, v]) => <Fragment key={k}><dt>{k}</dt><dd className={k.includes('template') || k.includes('context') ? 'mono' : ''}>{v}</dd></Fragment>)}</dl>
      </div>
    </Card>
  );
}

const LIMIT = 50;

export function AuditTab() {
  const [offset, setOffset] = useState(0);
  const { data, loading, error, reload } = useApi((s) => get<AuditEntry[]>('/audit', { limit: LIMIT + 1, offset }, s), [offset]);
  const rows = (data ?? []).slice(0, LIMIT);
  const hasMore = (data ?? []).length > LIMIT;
  return (
    <Card title="Audit log" icon={<ScrollText />} flush actions={<button className="btn sm" onClick={reload}><RefreshCw />Refresh</button>}>
      {error ? <div style={{ padding: 16 }}><ErrorBox error={error} onRetry={reload} /></div>
        : loading && !data ? <SkeletonRows rows={8} />
        : rows.length === 0 ? <Empty title="No audit entries" />
        : (
          <div className="table-wrap">
            <table className="table">
              <thead><tr><th>When</th><th>User</th><th>Action</th><th>Entity</th><th className="hide-sm">Detail</th></tr></thead>
              <tbody>
                {rows.map((a) => (
                  <tr key={a.id}>
                    <td className="tnum">{fmtDateTime(a.at)}</td>
                    <td>{a.username || (a.user_id ? `#${a.user_id}` : 'system')}</td>
                    <td><span className="badge">{a.action}</span></td>
                    <td>{a.entity}{a.entity_id ? <span className="muted"> #{a.entity_id}</span> : ''}</td>
                    <td className="hide-sm small" style={{ maxWidth: 420, wordBreak: 'break-word' }}>{a.detail}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      {(offset > 0 || hasMore) && <Pager offset={offset} limit={LIMIT} total={hasMore ? offset + LIMIT + 1 : offset + rows.length} onChange={setOffset} />}
    </Card>
  );
}
