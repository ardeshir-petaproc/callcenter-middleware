import { useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { BookUser, Pencil, Plus, Search, Star, Trash2 } from 'lucide-react';
import type { Contact } from '../../api/types';
import { del, get } from '../../api/client';
import { useApi, useDebounced } from '../../lib/hooks';
import { Card, Empty, ErrorBox, PageHeader, SkeletonRows } from '../../components/ui/primitives';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';
import { ContactDialog } from './ContactDialog';

export default function ContactsPage() {
  const [params, setParams] = useSearchParams();
  const [q, setQ] = useState('');
  const dq = useDebounced(q, 300);
  const { data, loading, error, reload, setData } = useApi((s) => get<Contact[]>('/contacts', { q: dq }, s), [dq]);
  const preset = params.get('new');
  const [edit, setEdit] = useState<Contact | null | 'new'>(preset ? 'new' : null);
  const [vipOnly, setVipOnly] = useState(false);

  const items = (data ?? []).filter((c) => !vipOnly || c.vip);
  const remove = async (c: Contact) => {
    if (!(await confirm({ title: 'Delete contact?', message: `Delete ${c.name || c.phone}?`, danger: true, confirmLabel: 'Delete' }))) return;
    try { await del(`/contacts/${c.id}`); setData((l) => l?.filter((x) => x.id !== c.id)); toast.success('Contact deleted'); } catch (e) { toast.error(e); }
  };
  const close = () => { setEdit(null); if (preset) setParams({}, { replace: true }); };

  return (
    <>
      <PageHeader title="Contacts" sub="Customer records used for screen-pops and routing" actions={<button className="btn primary" onClick={() => setEdit('new')}><Plus />New contact</button>} />
      <Card flush title={
        <div className="input-icon" style={{ width: 'min(320px, 70vw)' }}>
          <Search /><input className="input sm" placeholder="Search name, phone, company…" value={q} onChange={(e) => setQ(e.target.value)} aria-label="Search contacts" />
        </div>
      } actions={<label className="checkbox small"><input type="checkbox" checked={vipOnly} onChange={(e) => setVipOnly(e.target.checked)} />VIP only</label>}>
        {error ? <div style={{ padding: 16 }}><ErrorBox error={error} onRetry={reload} /></div>
          : loading && !data ? <SkeletonRows rows={6} />
          : items.length === 0 ? <Empty title={q ? 'No matches' : 'No contacts yet'} icon={<BookUser />} />
          : (
            <div className="table-wrap">
              <table className="table">
                <thead><tr><th>Name</th><th>Phone</th><th className="hide-sm">Company</th><th className="hide-sm">Email</th><th className="hide-sm">Tags</th><th /></tr></thead>
                <tbody>
                  {items.map((c) => (
                    <tr key={c.id} className="clickable" onClick={() => setEdit(c)}>
                      <td><div className="row" style={{ gap: 6 }}><span className="strong">{c.name || '—'}</span>{c.vip && <span className="badge warn"><Star size={11} />VIP</span>}</div></td>
                      <td className="tnum"><Link to={`/calls?q=${encodeURIComponent(c.phone)}`} onClick={(e) => e.stopPropagation()}>{c.phone}</Link></td>
                      <td className="hide-sm">{c.company || <span className="muted">—</span>}</td>
                      <td className="hide-sm">{c.email || <span className="muted">—</span>}</td>
                      <td className="hide-sm"><div className="chips">{(c.tags ?? []).map((t) => <span key={t} className="chip">{t}</span>)}</div></td>
                      <td className="actions" onClick={(e) => e.stopPropagation()}>
                        <button className="btn sm icon ghost" aria-label="Edit" onClick={() => setEdit(c)}><Pencil /></button>
                        <button className="btn sm icon ghost" aria-label="Delete" onClick={() => remove(c)}><Trash2 /></button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
      </Card>
      <ContactDialog open={edit !== null} initial={edit === 'new' ? null : edit} presetPhone={preset ?? undefined} onClose={close} onSaved={() => reload()} />
    </>
  );
}
