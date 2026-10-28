import { useState } from 'react';
import { History } from 'lucide-react';
import { get } from '../api/client';
import type { Revision, RevisionEntity } from '../api/types_org';
import { useApi } from '../lib/hooks';
import { PERMS } from '../lib/perms';
import { useCan } from '../state/auth';
import { fmtDateTime, humanize } from '../lib/format';
import { Drawer } from './ui/Dialog';
import { Empty, ErrorBox, SkeletonRows } from './ui/primitives';

/** "History" button opening a drawer with the change history of one configuration object. */
export function RevisionsButton({ entity, id, label = 'History' }: { entity: RevisionEntity; id: number; label?: string }) {
  const [open, setOpen] = useState(false);
  // The server only exposes revision history to org.manage.
  const allowed = useCan()(PERMS.orgManage);
  if (!id || !allowed) return null;
  return (
    <>
      <button type="button" className="btn ghost" onClick={() => setOpen(true)}><History />{label}</button>
      <Drawer open={open} onClose={() => setOpen(false)} title="Revision history">
        {open && <RevisionList entity={entity} id={id} />}
      </Drawer>
    </>
  );
}

function RevisionList({ entity, id }: { entity: RevisionEntity; id: number }) {
  const { data, loading, error, reload } = useApi((s) => get<Revision[]>('/revisions', { entity, entity_id: id }, s), [entity, id]);
  const [openId, setOpenId] = useState<number | null>(null);
  if (error) return <ErrorBox error={error} onRetry={reload} />;
  if (loading && !data) return <SkeletonRows rows={4} />;
  if (!data?.length) return <Empty title="No revisions recorded">Changes made from now on will appear here.</Empty>;
  return (
    <ul className="timeline">
      {data.map((r) => (
        <li key={r.id}>
          <span className="t-type">{humanize(r.action)}</span>
          <span className="t-time">{fmtDateTime(r.at)}</span>
          <div className="t-detail">by <span dir="auto">{r.changed_by_name || (r.changed_by ? `user #${r.changed_by}` : 'system')}</span>
            {' · '}<button className="btn ghost sm" style={{ height: 22 }} onClick={() => setOpenId(openId === r.id ? null : r.id)}>{openId === r.id ? 'Hide' : 'Show'} snapshot</button>
          </div>
          {openId === r.id && <pre className="code-block" dir="auto">{pretty(r.data)}</pre>}
        </li>
      ))}
    </ul>
  );
}

function pretty(raw: string): string {
  try { return JSON.stringify(JSON.parse(raw), null, 2); } catch { return raw; }
}
