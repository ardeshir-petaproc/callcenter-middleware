import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { BookOpen, FilePlus2, Search, Upload } from 'lucide-react';
import { get } from '../../api/client';
import type { KBCategory, KnowledgeSource } from '../../api/types_work';
import { useApi, useDebounced } from '../../lib/hooks';
import { fmtDateTime, humanize } from '../../lib/format';
import { PERMS } from '../../lib/perms';
import { useCan } from '../../state/auth';
import { Card, Empty, ErrorBox, PageHeader, SkeletonRows } from '../../components/ui/primitives';
import { KBBadges, fmtBytes } from './kbUi';
import { KBSourceDialog } from './KBSourceDialog';
import { KBViewDrawer } from './KBViewDrawer';
import { KBSearchPanel } from './KBSearchPanel';

export default function KnowledgePage() {
  const canManage = useCan()(PERMS.kbManage);
  const [params, setParams] = useSearchParams();
  const [q, setQ] = useState('');
  const dq = useDebounced(q, 300);
  const category = params.get('category') ?? '';
  const viewId = Number(params.get('source')) || 0;
  const list = useApi((s) => get<KnowledgeSource[]>('/kb/sources', { q: dq, category }, s), [dq, category]);
  const cats = useApi((s) => get<KBCategory[]>('/kb/categories', undefined, s), []);
  const [edit, setEdit] = useState<{ mode: 'file' | 'text'; source: KnowledgeSource | null } | null>(null);
  const setParam = (k: string, v: string) => { const n = new URLSearchParams(params); if (v) n.set(k, v); else n.delete(k); setParams(n, { replace: true }); };
  const reloadAll = () => { list.reload(); cats.reload(); };
  useEffect(() => { if (viewId && list.data && !list.data.some((s) => s.id === viewId)) list.reload(); }, [viewId]); // eslint-disable-line react-hooks/exhaustive-deps

  const items = list.data ?? [];
  return (
    <>
      <PageHeader title="Knowledge base" sub="Documents, notes and emails your team and the AI can rely on"
        actions={canManage && <>
          <button className="btn" onClick={() => setEdit({ mode: 'text', source: null })}><FilePlus2 />New text</button>
          <button className="btn primary" onClick={() => setEdit({ mode: 'file', source: null })}><Upload />Upload file</button>
        </>} />
      <KBSearchPanel canManage={canManage} onOpen={(id) => setParam('source', String(id))} />
      <Card flush>
        <div className="row wrap" style={{ gap: 8, padding: '12px 16px', borderBottom: '1px solid var(--border)' }}>
          <div className="input-icon" style={{ flex: '1 1 200px', maxWidth: 300 }}>
            <Search /><input className="input sm" dir="auto" placeholder="Filter by title or tag…" value={q} onChange={(e) => setQ(e.target.value)} aria-label="Filter sources" />
          </div>
          <select className="select sm" style={{ width: 180 }} value={category} onChange={(e) => setParam('category', e.target.value)} aria-label="Category">
            <option value="">All categories</option>
            {(cats.data ?? []).map((c) => <option key={c.category} value={c.category}>{c.category || '(none)'} ({c.count})</option>)}
          </select>
        </div>
        {list.error ? <div style={{ padding: 16 }}><ErrorBox error={list.error} onRetry={list.reload} /></div>
          : list.loading && !list.data ? <SkeletonRows rows={6} />
          : items.length === 0 ? <Empty title="No knowledge sources" icon={<BookOpen />}>{canManage ? 'Upload a document or write a note to get started.' : 'Nothing has been shared with you yet.'}</Empty>
          : (
            <div className="table-wrap">
              <table className="table">
                <thead><tr><th>Title</th><th>Category</th><th className="hide-sm">Type</th><th>Status</th><th className="hide-sm">Valid until</th><th className="hide-sm">Updated</th></tr></thead>
                <tbody>
                  {items.map((s) => (
                    <tr key={s.id} className="clickable" onClick={() => setParam('source', String(s.id))}>
                      <td style={{ maxWidth: 360 }}>
                        <div className="strong ellipsis" dir="auto">{s.title}</div>
                        <div className="chips">{(s.tags ?? []).map((t) => <span key={t} className="chip" dir="auto">{t}</span>)}</div>
                      </td>
                      <td dir="auto">{s.category || <span className="muted">—</span>}</td>
                      <td className="hide-sm">{humanize(s.kind)}{s.kind === 'file' ? ` · ${fmtBytes(s.size)}` : ''}</td>
                      <td><KBBadges s={s} />{s.status === 'active' && <span className="small muted">Active</span>}</td>
                      <td className="hide-sm">{s.valid_until || '—'}</td>
                      <td className="hide-sm tnum">{fmtDateTime(s.updated_at)} · v{s.version}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
      </Card>
      <KBViewDrawer id={viewId} canManage={canManage} onClose={() => setParam('source', '')}
        onEdit={(s) => setEdit({ mode: s.kind === 'text' ? 'text' : 'file', source: s })} onDeleted={() => { setParam('source', ''); reloadAll(); }} />
      <KBSourceDialog open={!!edit} mode={edit?.mode ?? 'file'} source={edit?.source ?? null} categories={(cats.data ?? []).map((c) => c.category).filter(Boolean)}
        onClose={() => setEdit(null)} onSaved={(s) => { reloadAll(); setParam('source', String(s.id)); }} />
    </>
  );
}
