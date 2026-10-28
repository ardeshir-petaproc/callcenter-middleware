import { useEffect, useState } from 'react';
import { get, post, put, upload } from '../../api/client';
import type { KnowledgeSource } from '../../api/types_work';
import { Dialog } from '../../components/ui/Dialog';
import { Field } from '../../components/ui/primitives';
import { toast } from '../../components/ui/Toast';
import { KBMetaFields, type KBMeta } from './KBMetaFields';
import { KB_ACCEPT, fmtBytes } from './kbUi';

const blank: KBMeta = { title: '', category: '', tags: [], department_ids: [], visibility: 'all', allowed_roles: [], ai_enabled: true, status: 'active', valid_until: '', topics: [] };
const TEXT_LIMIT = 20000; // the server returns at most this many characters of extracted text

interface Props { open: boolean; mode: 'file' | 'text'; source: KnowledgeSource | null; categories: string[]; onClose: () => void; onSaved: (s: KnowledgeSource) => void }

/** Create (upload a file or write text) or edit a knowledge source's metadata and content. */
export function KBSourceDialog({ open, mode, source, categories, onClose, onSaved }: Props) {
  const [meta, setMeta] = useState<KBMeta>(blank);
  const [file, setFile] = useState<File | null>(null);
  const [text, setText] = useState('');
  const [origText, setOrigText] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    if (!open) return;
    setFile(null); setText(''); setOrigText(null);
    setMeta(source ? {
      title: source.title, category: source.category, tags: source.tags ?? [], department_ids: source.department_ids ?? [], visibility: source.visibility || 'all',
      allowed_roles: source.allowed_roles ?? [], ai_enabled: source.ai_enabled, status: source.status || 'active', valid_until: source.valid_until ?? '', topics: source.topics ?? [],
    } : blank);
    if (source && mode === 'text') {
      get<{ text: string }>(`/kb/sources/${source.id}`).then((r) => { setText(r.text ?? ''); setOrigText(r.text ?? ''); }).catch(() => setOrigText(''));
    }
  }, [open, source, mode]);
  const set = (p: Partial<KBMeta>) => setMeta((m) => ({ ...m, ...p }));
  const truncated = origText !== null && origText.length >= TEXT_LIMIT;
  const valid = meta.title.trim() && (source || (mode === 'file' ? !!file : !!text.trim()));

  const save = async () => {
    setBusy(true);
    try {
      const body = { ...(source ?? {}), ...meta, kind: source?.kind ?? mode };
      let saved: KnowledgeSource;
      if (file) {
        const fd = new FormData();
        fd.set('meta', JSON.stringify(body));
        fd.set('file', file);
        saved = await upload<KnowledgeSource>(source ? `/kb/sources/${source.id}` : '/kb/sources', fd, source ? 'PUT' : 'POST');
      } else {
        const content = mode === 'text' && !truncated && text !== origText ? text : undefined;
        saved = source ? await put<KnowledgeSource>(`/kb/sources/${source.id}`, { ...body, content }) : await post<KnowledgeSource>('/kb/sources', { ...body, content: text });
      }
      toast.success(source ? 'Knowledge source saved' : 'Knowledge source added');
      onSaved(saved);
      onClose();
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };

  return (
    <Dialog open={open} onClose={onClose} size="wide" title={source ? `Edit “${source.title}”` : mode === 'file' ? 'Upload a document' : 'New text source'}
      footer={<><button className="btn" onClick={onClose}>Cancel</button><button className="btn primary" disabled={busy || !valid} onClick={save}>{busy ? <span className="spinner" /> : null}{source ? 'Save' : 'Add'}</button></>}>
      {mode === 'file' ? (
        <Field label={source ? 'Replace file (optional)' : 'File'} hint="txt, md, csv, json, html, docx, pptx, xlsx or pdf (text layer) — max 25 MB; stored encrypted">
          <input className="input" type="file" accept={KB_ACCEPT} onChange={(e) => {
            const f = e.target.files?.[0] ?? null;
            setFile(f);
            if (f && !meta.title) set({ title: f.name.replace(/\.[^.]+$/, '') });
          }} />
          {source?.file_name && !file && <span className="small muted">Current: {source.file_name} ({fmtBytes(source.size)})</span>}
        </Field>
      ) : (
        <Field label="Content" hint={truncated ? 'This text is too long to edit here; content is kept unchanged.' : undefined}>
          <textarea className="textarea" dir="auto" style={{ minHeight: 180 }} value={text} disabled={truncated} onChange={(e) => setText(e.target.value)} placeholder="Write the knowledge here, e.g. a policy or an FAQ answer…" />
        </Field>
      )}
      <KBMetaFields m={meta} set={set} categories={categories} />
    </Dialog>
  );
}
