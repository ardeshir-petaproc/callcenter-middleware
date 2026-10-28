import { useState } from 'react';
import { FolderSearch } from 'lucide-react';
import { post } from '../../api/client';
import type { EmailConnector, EmailSecurity } from '../../api/types_work';
import { useLookup } from '../../state/lookups';
import { Field, Segmented, Toggle } from '../../components/ui/primitives';
import { TagInput } from '../../components/ui/TagInput';
import { toast } from '../../components/ui/Toast';

export type Draft = Partial<EmailConnector> & { password: string };
interface StepProps { c: Draft; set: (p: Partial<Draft>) => void }

const SECURITY: { value: EmailSecurity; label: string }[] = [{ value: 'tls', label: 'TLS' }, { value: 'starttls', label: 'STARTTLS' }, { value: 'none', label: 'None (test only)' }];

export function MailboxStep({ c, set }: StepProps) {
  const [folders, setFolders] = useState<string[] | null>(null);
  const [busy, setBusy] = useState(false);
  const load = async () => {
    setBusy(true);
    try {
      const list = await post<string[]>('/email-connectors/folders', { ...c, id: c.id || undefined });
      setFolders(list ?? []);
      if (list?.length && !list.includes(c.folder ?? '')) set({ folder: list.includes('INBOX') ? 'INBOX' : list[0] });
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  return (
    <>
      <div className="form-grid">
        <Field label="Name"><input className="input" dir="auto" placeholder="Support inbox" value={c.name ?? ''} onChange={(e) => set({ name: e.target.value })} data-autofocus /></Field>
        <Field label="IMAP host"><input className="input mono" placeholder="imap.example.com" value={c.host ?? ''} onChange={(e) => set({ host: e.target.value })} /></Field>
        <Field label="Port" hint="Default 993 (TLS) or 143"><input className="input" type="number" min={0} value={c.port || ''} onChange={(e) => set({ port: Number(e.target.value) || 0 })} /></Field>
        <Field label="Security">
          <select className="select" value={c.security ?? 'tls'} onChange={(e) => set({ security: e.target.value as EmailSecurity })}>
            {SECURITY.map((s) => <option key={s.value} value={s.value}>{s.label}</option>)}
          </select>
        </Field>
        <Field label="Username"><input className="input" autoComplete="off" value={c.username ?? ''} onChange={(e) => set({ username: e.target.value })} /></Field>
        <Field label="Password" hint={c.has_password ? 'Leave empty to keep the stored password' : 'Stored encrypted, never shown again'}>
          <input className="input" type="password" autoComplete="new-password" value={c.password} onChange={(e) => set({ password: e.target.value })} />
        </Field>
      </div>
      <div className="form-section">
        <h3>Folder to share</h3>
        <p className="small muted">Only the selected folder is read — never the rest of the mailbox.</p>
        <div className="row wrap">
          {folders?.length ? (
            <select className="select" style={{ maxWidth: 320 }} value={c.folder ?? ''} onChange={(e) => set({ folder: e.target.value })} aria-label="Folder">
              {folders.map((f) => <option key={f} value={f}>{f}</option>)}
            </select>
          ) : <input className="input" style={{ maxWidth: 320 }} value={c.folder ?? 'INBOX'} onChange={(e) => set({ folder: e.target.value })} aria-label="Folder" />}
          <button type="button" className="btn" disabled={busy || !c.host || !c.username || (!c.password && !c.has_password)} onClick={load}>
            {busy ? <span className="spinner" /> : <FolderSearch />}Load folders
          </button>
        </div>
      </div>
    </>
  );
}

export function ImportStep({ c, set }: StepProps) {
  const depts = useLookup('departments');
  return (
    <>
      <Segmented label="Mode" value={c.mode ?? 'ticket'} onChange={(m) => set({ mode: m })}
        options={[{ value: 'ticket', label: 'Create tickets' }, { value: 'kb', label: 'Add to knowledge base' }]} />
      <p className="small muted">{c.mode === 'kb' ? 'Each message becomes a draft knowledge source until reviewed.' : 'Each new conversation opens a ticket; replies are added to it.'}</p>
      <div className="form-grid">
        <div className="field span-all"><span className="field-label">Only from these senders</span>
          <TagInput value={c.from_filter ?? []} onChange={(v) => set({ from_filter: v })} placeholder="name@example.com or @example.com — empty = any sender" /></div>
        <Field label="Subject contains"><input className="input" dir="auto" value={c.subject_filter ?? ''} onChange={(e) => set({ subject_filter: e.target.value })} /></Field>
        <Field label="Ignore mail before" hint="Older messages are never imported"><input className="input" type="date" value={c.since_date ?? ''} onChange={(e) => set({ since_date: e.target.value })} /></Field>
        <Field label="Department">
          <select className="select" value={c.department_id || ''} onChange={(e) => set({ department_id: Number(e.target.value) || 0 })}>
            <option value="">None</option>
            {depts.map((d) => <option key={d.id} value={d.id}>{d.name}</option>)}
          </select>
        </Field>
        {c.mode === 'kb'
          ? <Field label="Knowledge category"><input className="input" dir="auto" value={c.kb_category ?? ''} onChange={(e) => set({ kb_category: e.target.value })} /></Field>
          : <Field label="Ticket category"><input className="input" dir="auto" value={c.ticket_category ?? ''} onChange={(e) => set({ ticket_category: e.target.value })} /></Field>}
        <Field label="Check every (minutes)"><input className="input" type="number" min={1} max={1440} value={c.poll_minutes || 5} onChange={(e) => set({ poll_minutes: Number(e.target.value) })} /></Field>
      </div>
      <div className="col">
        <Toggle checked={!!c.ai_enabled} onChange={(v) => set({ ai_enabled: v })} label="Allow the AI to use knowledge imported from this mailbox" />
        <Toggle checked={c.enabled ?? true} onChange={(v) => set({ enabled: v })} label="Sync automatically" />
      </div>
    </>
  );
}

export function SmtpStep({ c, set }: StepProps) {
  return (
    <>
      <p className="small muted">Optional. When set, public ticket replies are emailed to the customer from this address. Uses the same username and password.</p>
      <div className="form-grid">
        <Field label="SMTP host"><input className="input mono" placeholder="smtp.example.com" value={c.smtp_host ?? ''} onChange={(e) => set({ smtp_host: e.target.value })} /></Field>
        <Field label="SMTP port" hint="e.g. 465 (TLS) or 587 (STARTTLS)"><input className="input" type="number" min={0} value={c.smtp_port || ''} onChange={(e) => set({ smtp_port: Number(e.target.value) || 0 })} /></Field>
        <Field label="SMTP security">
          <select className="select" value={c.smtp_security || 'starttls'} onChange={(e) => set({ smtp_security: e.target.value as EmailSecurity })}>
            {SECURITY.map((s) => <option key={s.value} value={s.value}>{s.label}</option>)}
          </select>
        </Field>
        <Field label="From address"><input className="input" type="email" placeholder="support@example.com" value={c.from_address ?? ''} onChange={(e) => set({ from_address: e.target.value })} /></Field>
      </div>
    </>
  );
}
