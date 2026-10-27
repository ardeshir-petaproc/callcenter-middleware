import { useMemo, useState } from 'react';
import { Upload } from 'lucide-react';
import { post } from '../../api/client';
import { Dialog } from '../../components/ui/Dialog';
import { toast } from '../../components/ui/Toast';
import { parseCsv } from './campaignUi';

interface Props { campaignId: number; open: boolean; onClose: () => void; onImported: () => void }

const HEADER = /^(number|phone|tel|telephone|mobile)$/i;

/** Paste or upload a CSV (number,name,notes), preview it client-side, then POST the parsed leads. */
export function LeadImportDialog({ campaignId, open, onClose, onImported }: Props) {
  const [text, setText] = useState('');
  const [busy, setBusy] = useState(false);
  const leads = useMemo(() => {
    const rows = parseCsv(text);
    if (rows.length && HEADER.test(rows[0][0] ?? '')) rows.shift();
    return rows.filter((r) => r[0]).map(([number, name = '', notes = '']) => ({ number, name, notes }));
  }, [text]);

  const onFile = async (f: File | undefined) => {
    if (!f) return;
    if (f.size > 5_000_000) { toast.error('File is larger than 5 MB'); return; }
    setText(await f.text());
  };
  const submit = async () => {
    setBusy(true);
    try {
      const res = await post<{ added: number; skipped: number }>(`/campaigns/${campaignId}/leads`, { leads });
      toast.success(`Imported ${res?.added ?? leads.length} lead(s)${res?.skipped ? `, skipped ${res.skipped} duplicate/invalid` : ''}`);
      setText('');
      onImported();
      onClose();
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };

  return (
    <Dialog open={open} onClose={onClose} title="Import leads" size="wide"
      footer={<><button className="btn" onClick={onClose}>Cancel</button>
        <button className="btn primary" disabled={busy || !leads.length} onClick={submit}><Upload />Import {leads.length || ''} lead{leads.length === 1 ? '' : 's'}</button></>}>
      <p className="small muted">One lead per line: <code>number,name,notes</code>. A header row is detected automatically. Commas, semicolons and tabs are accepted.</p>
      <label className="btn" style={{ alignSelf: 'flex-start' }}>
        <Upload />Choose CSV file
        <input type="file" accept=".csv,text/csv,text/plain" className="sr-only" onChange={(e) => void onFile(e.target.files?.[0])} />
      </label>
      <textarea className="textarea mono" style={{ minHeight: 160 }} placeholder={'number,name,notes\n+15550123,Jane Doe,Renewal due in Nov'} value={text} onChange={(e) => setText(e.target.value)} aria-label="CSV data" />
      {leads.length > 0 && (
        <div className="table-wrap" style={{ border: '1px solid var(--border)', borderRadius: 8, maxHeight: 200 }}>
          <table className="table">
            <thead><tr><th>Number</th><th>Name</th><th>Notes</th></tr></thead>
            <tbody>
              {leads.slice(0, 5).map((l, i) => <tr key={i}><td className="num">{l.number}</td><td>{l.name || '—'}</td><td className="small">{l.notes || '—'}</td></tr>)}
            </tbody>
          </table>
          {leads.length > 5 && <div className="pager">…and {leads.length - 5} more</div>}
        </div>
      )}
    </Dialog>
  );
}
