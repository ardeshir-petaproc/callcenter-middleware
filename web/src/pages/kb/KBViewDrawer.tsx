import { Fragment, type ReactNode } from 'react';
import { Download, Pencil, Trash2 } from 'lucide-react';
import { authUrl, del, get } from '../../api/client';
import type { KnowledgeSource } from '../../api/types_work';
import { useApi } from '../../lib/hooks';
import { fmtDateTime, humanize } from '../../lib/format';
import { useDepartmentNames, useLookup, useUserNames } from '../../state/lookups';
import { Drawer } from '../../components/ui/Dialog';
import { ErrorBox, SkeletonRows } from '../../components/ui/primitives';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';
import { KBBadges, fmtBytes } from './kbUi';

interface Props { id: number; refreshKey: number; canManage: boolean; onClose: () => void; onEdit: (s: KnowledgeSource) => void; onDeleted: () => void }

/** Details, extracted text and actions for one knowledge source. */
export function KBViewDrawer({ id, refreshKey, canManage, onClose, onEdit, onDeleted }: Props) {
  const { data, loading, error, reload } = useApi((s) => get<{ source: KnowledgeSource; text: string }>(`/kb/sources/${id}`, undefined, s), [id, refreshKey], id > 0);
  const deptName = useDepartmentNames();
  const userName = useUserNames();
  const roles = useLookup('customRoles');
  const src = data?.source;
  const roleLabel = (r: string) => (r.startsWith('role:') ? roles.find((x) => `role:${x.id}` === r)?.name ?? r : humanize(r));

  const remove = async () => {
    if (!src || !(await confirm({ title: 'Delete knowledge source?', message: `Delete “${src.title}” and its stored file? The AI will no longer use it.`, danger: true, confirmLabel: 'Delete' }))) return;
    try { await del(`/kb/sources/${src.id}`); toast.success('Deleted'); onDeleted(); } catch (e) { toast.error(e); }
  };
  const meta: [string, ReactNode][] = src ? [
    ['Status', <span className="row wrap" style={{ gap: 4 }}>{humanize(src.status)}<KBBadges s={src} /></span>],
    ['Category', <span dir="auto">{src.category || '—'}</span>], ['Type', humanize(src.kind)],
    ['File', src.file_name ? `${src.file_name} · ${fmtBytes(src.size)}` : '—'],
    ['Departments', (src.department_ids ?? []).map(deptName).join(', ') || 'Whole organization'],
    ['Visibility', src.visibility === 'restricted' ? `Restricted: ${(src.allowed_roles ?? []).map(roleLabel).join(', ') || 'kb.restricted only'}` : 'Every employee'],
    ['AI use', src.ai_enabled ? 'Allowed' : 'Not allowed'], ['Valid until', src.valid_until || 'No expiry'],
    ['Tags', (src.tags ?? []).join(', ') || '—'], ['Owner', userName(src.owner_id)],
    ['Version', `v${src.version} · ${fmtDateTime(src.updated_at)}`], ['Extracted', `${src.text_chars.toLocaleString()} characters`],
  ] : [];

  return (
    <Drawer open={id > 0} onClose={onClose} title={src?.title ?? 'Knowledge source'} actions={src && (
      <>
        <a className="btn sm" href={authUrl(`/kb/sources/${src.id}/download`)} download><Download />Download</a>
        {canManage && <button className="btn sm icon" aria-label="Edit" onClick={() => onEdit(src)}><Pencil /></button>}
        {canManage && <button className="btn sm icon" aria-label="Delete" onClick={remove}><Trash2 /></button>}
      </>
    )}>
      {error ? <ErrorBox error={error} onRetry={reload} />
        : loading && !data ? <SkeletonRows rows={6} />
        : src && (
          <div className="col gap-lg">
            <dl className="kv">{meta.map(([k, v]) => <Fragment key={k}><dt>{k}</dt><dd>{v}</dd></Fragment>)}</dl>
            <div className="form-section">
              <h3>Extracted text</h3>
              {data?.text ? <div className="text-view" dir="auto">{data.text}</div> : <span className="small muted">No text could be extracted.</span>}
            </div>
          </div>
        )}
    </Drawer>
  );
}
