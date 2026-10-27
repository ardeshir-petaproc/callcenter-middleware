import { useState, type ReactNode } from 'react';
import { useNavigate } from 'react-router-dom';
import { Plus, Search } from 'lucide-react';
import { Empty, ErrorBox, PageHeader, SkeletonRows } from './ui/primitives';

interface Props<T extends { id: number }> {
  title: string;
  sub?: ReactNode;
  basePath: string;
  items: T[];
  loading: boolean;
  error: string | null;
  onRetry: () => void;
  selected: string | undefined;
  canCreate: boolean;
  newLabel?: string;
  search: (item: T) => string;
  renderItem: (item: T) => ReactNode;
  children: ReactNode;
  headerActions?: ReactNode;
  emptyIcon?: ReactNode;
}

/** Master/detail layout for configuration entities: searchable list on the left, editor on the right. */
export function ConfigLayout<T extends { id: number }>(p: Props<T>) {
  const nav = useNavigate();
  const [q, setQ] = useState('');
  const needle = q.trim().toLowerCase();
  const items = needle ? p.items.filter((i) => p.search(i).toLowerCase().includes(needle)) : p.items;
  return (
    <>
      <PageHeader title={p.title} sub={p.sub} actions={
        <>
          {p.headerActions}
          {p.canCreate && <button className="btn primary" onClick={() => nav(`${p.basePath}/new`)}><Plus />{p.newLabel ?? 'New'}</button>}
        </>
      } />
      <div className="split">
        <div className="card sticky-list">
          {p.items.length > 6 && (
            <div style={{ padding: 10, borderBottom: '1px solid var(--border)' }}>
              <div className="input-icon"><Search /><input className="input sm" placeholder="Filter…" value={q} onChange={(e) => setQ(e.target.value)} aria-label="Filter list" /></div>
            </div>
          )}
          {p.error ? <div style={{ padding: 12 }}><ErrorBox error={p.error} onRetry={p.onRetry} /></div>
            : p.loading && !p.items.length ? <SkeletonRows rows={5} />
            : items.length === 0 ? <Empty title={needle ? 'No matches' : 'Nothing here yet'} icon={p.emptyIcon} />
            : (
              <div className="list" role="list">
                {items.map((i) => (
                  <div key={i.id} role="listitem" tabIndex={0}
                    className={`list-item clickable ${String(i.id) === p.selected ? 'selected' : ''}`}
                    onClick={() => nav(`${p.basePath}/${i.id}`)}
                    onKeyDown={(e) => { if (e.key === 'Enter') nav(`${p.basePath}/${i.id}`); }}>
                    {p.renderItem(i)}
                  </div>
                ))}
              </div>
            )}
        </div>
        <div className="col" style={{ gap: 16, minWidth: 0 }}>{p.children}</div>
      </div>
    </>
  );
}

export function EditorFooter({ onSave, onDelete, busy, canEdit, dirty, isNew }: {
  onSave: () => void; onDelete?: () => void; busy: boolean; canEdit: boolean; dirty: boolean; isNew: boolean;
}) {
  if (!canEdit) return <div className="card-foot"><span className="small muted">Read-only — admin access is required to edit.</span></div>;
  return (
    <div className="card-foot" style={{ justifyContent: 'space-between' }}>
      <div>{!isNew && onDelete && <button type="button" className="btn ghost" style={{ color: 'var(--danger)' }} onClick={onDelete}>Delete</button>}</div>
      <div className="row">
        {dirty && <span className="small muted">Unsaved changes</span>}
        <button type="button" className="btn primary" disabled={busy || (!dirty && !isNew)} onClick={onSave}>{isNew ? 'Create' : 'Save changes'}</button>
      </div>
    </div>
  );
}
