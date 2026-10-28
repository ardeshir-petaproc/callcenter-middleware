import { useEffect, useState } from 'react';
import { ShieldCheck } from 'lucide-react';
import { post, put } from '../../api/client';
import type { EmailConnector } from '../../api/types_work';
import { Dialog } from '../../components/ui/Dialog';
import { Tabs } from '../../components/ui/primitives';
import { toast } from '../../components/ui/Toast';
import { ImportStep, MailboxStep, SmtpStep, type Draft } from './EmailSteps';

const blank: Draft = {
  name: '', host: '', port: 0, security: 'tls', username: '', password: '', folder: 'INBOX', from_filter: [], subject_filter: '',
  since_date: '', mode: 'ticket', department_id: 0, ticket_category: '', kb_category: '', ai_enabled: false, poll_minutes: 5,
  enabled: true, smtp_host: '', smtp_port: 0, smtp_security: undefined, from_address: '',
};
type Step = 'mailbox' | 'import' | 'smtp' | 'consent';
const STEPS: { value: Step; label: string }[] = [
  { value: 'mailbox', label: '1. Mailbox' }, { value: 'import', label: '2. What to import' },
  { value: 'smtp', label: '3. Replies (optional)' }, { value: 'consent', label: '4. Consent' },
];

/** Connect (wizard with mandatory consent) or edit a scoped IMAP mailbox connection. */
export function EmailConnectorDialog({ open, initial, onClose, onSaved }: {
  open: boolean; initial: EmailConnector | null; onClose: () => void; onSaved: (c: EmailConnector) => void;
}) {
  const [c, setC] = useState<Draft>(blank);
  const [step, setStep] = useState<Step>('mailbox');
  const [consent, setConsent] = useState(false);
  const [busy, setBusy] = useState(false);
  const isNew = !initial;
  useEffect(() => {
    if (!open) return;
    setC(initial ? { ...initial, password: '', from_filter: initial.from_filter ?? [] } : blank);
    setStep('mailbox');
    setConsent(false);
  }, [open, initial]);
  const set = (p: Partial<Draft>) => setC((x) => ({ ...x, ...p }));
  const steps = isNew ? STEPS : STEPS.filter((s) => s.value !== 'consent');
  const idx = steps.findIndex((s) => s.value === step);
  const mailboxOk = !!(c.host && c.username && (c.password || c.has_password) && c.folder);

  const save = async () => {
    setBusy(true);
    try {
      const body = { ...c, smtp_host: c.smtp_host?.trim() || '', consent: isNew ? consent : undefined };
      const saved = isNew ? await post<EmailConnector>('/email-connectors', body) : await put<EmailConnector>(`/email-connectors/${c.id}`, body);
      toast.success(isNew ? 'Mailbox connected' : 'Connector saved');
      onSaved(saved);
      onClose();
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };

  const last = idx === steps.length - 1;
  return (
    <Dialog open={open} onClose={onClose} title={isNew ? 'Connect a mailbox' : `Edit ${initial?.name || 'connector'}`} size="wide"
      footer={<>
        <button className="btn" onClick={idx > 0 ? () => setStep(steps[idx - 1].value) : onClose}>{idx > 0 ? 'Back' : 'Cancel'}</button>
        {!last && <button className="btn primary" disabled={step === 'mailbox' && !mailboxOk} onClick={() => setStep(steps[idx + 1].value)}>Next</button>}
        {(last || !isNew) && <button className="btn primary" disabled={busy || !mailboxOk || (isNew && !consent)} onClick={save}>{isNew ? 'Connect mailbox' : 'Save'}</button>}
      </>}>
      <Tabs value={step} tabs={steps} onChange={(s) => (mailboxOk || s === 'mailbox') && setStep(s)} />
      {step === 'mailbox' && <MailboxStep c={c} set={set} />}
      {step === 'import' && <ImportStep c={c} set={set} />}
      {step === 'smtp' && <SmtpStep c={c} set={set} />}
      {step === 'consent' && (
        <div className="col gap-lg">
          <div className="alert info"><ShieldCheck /><div>
            <strong>Only the folder “{c.folder}” of {c.username} is read</strong>{c.since_date ? `, starting from ${c.since_date}` : ''}
            {(c.from_filter ?? []).length ? `, only from ${(c.from_filter ?? []).join(', ')}` : ''}{c.subject_filter ? `, only with “${c.subject_filter}” in the subject` : ''}.
            {' '}Messages are stored encrypted and become {c.mode === 'kb' ? 'draft knowledge sources' : 'tickets'}.
            {c.ai_enabled ? ' The AI may use knowledge from this mailbox.' : ' The AI will not use this mailbox unless you allow it later.'}
            {' '}You can withdraw AI access or disconnect (and purge imported mail) at any time.
          </div></div>
          <label className="checkbox consent-box">
            <input type="checkbox" checked={consent} onChange={(e) => setConsent(e.target.checked)} />
            <span>I am authorized to connect this mailbox and I consent to the import described above.</span>
          </label>
        </div>
      )}
    </Dialog>
  );
}
