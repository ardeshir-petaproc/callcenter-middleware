import { useState } from 'react';
import { Bot, BotOff, Inbox, Mail, Pencil, Plus, RefreshCw, Unplug } from 'lucide-react';
import { del, post } from '../../api/client';
import type { EmailConnector } from '../../api/types_work';
import { useCrud } from '../../lib/crud';
import { fmtDateTime, fmtNum, humanize } from '../../lib/format';
import { Card, Empty, ErrorBox, SkeletonRows } from '../../components/ui/primitives';
import { Dialog } from '../../components/ui/Dialog';
import { toast } from '../../components/ui/Toast';
import { EmailConnectorDialog } from './EmailConnectorDialog';
import { EmailMessagesDialog } from './EmailMessagesDialog';

const tone: Record<string, string> = { ok: 'ok', error: 'danger', never: '' };

export function EmailTab() {
  const crud = useCrud<EmailConnector>('/email-connectors');
  const [edit, setEdit] = useState<EmailConnector | null | 'new'>(null);
  const [messages, setMessages] = useState<EmailConnector | null>(null);
  const [disconnect, setDisconnect] = useState<EmailConnector | null>(null);
  const [purge, setPurge] = useState(false);
  const [busy, setBusy] = useState<number | null>(null);

  const act = async (c: EmailConnector, fn: () => Promise<string>) => {
    setBusy(c.id);
    try { toast.success(await fn()); crud.reload(); } catch (e) { toast.error(e); crud.reload(); } finally { setBusy(null); }
  };
  const sync = (c: EmailConnector) => act(c, async () => `Imported ${(await post<{ imported: number }>(`/email-connectors/${c.id}/sync`)).imported} message(s)`);
  const ai = (c: EmailConnector, enabled: boolean) => act(c, async () => {
    const r = await post<{ sources_changed: number }>(`/email-connectors/${c.id}/ai`, { enabled });
    return `${enabled ? 'AI access allowed' : 'AI access withdrawn'} (${r.sources_changed} knowledge source(s) updated)`;
  });
  const doDisconnect = async () => {
    if (!disconnect) return;
    const c = disconnect;
    setDisconnect(null);
    await act(c, async () => { await del(`/email-connectors/${c.id}${purge ? '?purge=1' : ''}`); return purge ? 'Disconnected and purged' : 'Disconnected'; });
  };

  return (
    <Card title="Email connectors" icon={<Mail />} flush actions={<button className="btn sm primary" onClick={() => setEdit('new')}><Plus />Connect mailbox</button>}>
      {crud.error ? <div style={{ padding: 16 }}><ErrorBox error={crud.error} onRetry={crud.reload} /></div>
        : crud.loading && !crud.data ? <SkeletonRows rows={3} />
        : crud.items.length === 0 ? <Empty title="No mailboxes connected" icon={<Mail />}>Connect one folder of a mailbox to turn emails into tickets or knowledge — with explicit consent.</Empty>
        : (
          <div className="list">
            {crud.items.map((c) => (
              <div key={c.id} className="list-item" style={{ alignItems: 'flex-start', flexWrap: 'wrap' }}>
                <div className="grow" style={{ minWidth: 220 }}>
                  <div className="row wrap"><span className="strong" dir="auto">{c.name}</span><span className={`badge ${tone[c.status] ?? ''}`}>{c.status === 'never' ? 'Never synced' : humanize(c.status)}</span>
                    <span className="badge info">{c.mode === 'kb' ? 'Knowledge' : 'Tickets'}</span>{c.ai_enabled && <span className="badge violet">AI allowed</span>}{!c.enabled && <span className="badge">Paused</span>}</div>
                  <div className="small muted">{c.username} · folder <strong>{c.folder}</strong> · every {c.poll_minutes} min · {fmtNum(c.imported)} imported · last sync {fmtDateTime(c.last_sync)}</div>
                  {c.last_error && <div className="small" style={{ color: 'var(--danger)' }}>{c.last_error}</div>}
                  <div className="small muted">Consent given {fmtDateTime(c.consent_at)}{c.smtp_host ? ` · replies sent from ${c.from_address || c.username}` : ''}</div>
                </div>
                <div className="btn-group">
                  <button className="btn sm" disabled={busy === c.id} onClick={() => sync(c)}><RefreshCw />Sync now</button>
                  {c.ai_enabled ? <button className="btn sm" disabled={busy === c.id} onClick={() => ai(c, false)}><BotOff />Withdraw AI access</button>
                    : <button className="btn sm" disabled={busy === c.id} onClick={() => ai(c, true)}><Bot />Allow AI access</button>}
                  <button className="btn sm" onClick={() => setMessages(c)}><Inbox />Imported</button>
                  <button className="btn sm icon" aria-label="Edit" onClick={() => setEdit(c)}><Pencil /></button>
                  <button className="btn sm" style={{ color: 'var(--danger)' }} onClick={() => { setPurge(false); setDisconnect(c); }}><Unplug />Disconnect</button>
                </div>
              </div>
            ))}
          </div>
        )}
      <EmailConnectorDialog open={edit !== null} initial={edit === 'new' ? null : edit} onClose={() => setEdit(null)} onSaved={() => crud.reload()} />
      <EmailMessagesDialog connector={messages} onClose={() => setMessages(null)} />
      <Dialog open={!!disconnect} size="narrow" title="Disconnect mailbox?" onClose={() => setDisconnect(null)}
        footer={<><button className="btn" onClick={() => setDisconnect(null)}>Cancel</button><button className="btn danger" onClick={doDisconnect}>Disconnect</button></>}>
        <p>The stored credentials of <strong>{disconnect?.username}</strong> are deleted and no more mail is read.</p>
        <label className="checkbox"><input type="checkbox" checked={purge} onChange={(e) => setPurge(e.target.checked)} />Also delete all imported messages and knowledge from this mailbox</label>
      </Dialog>
    </Card>
  );
}
