import { useState } from 'react';
import { Pencil, Plus, Trash2, Webhook as WebhookIcon } from 'lucide-react';
import { EVENT_TYPES, type Webhook } from '../../api/types';
import { useCrud } from '../../lib/crud';
import { Card, Empty, ErrorBox, Field, SkeletonRows, Toggle } from '../../components/ui/primitives';
import { Dialog } from '../../components/ui/Dialog';
import { ChipSelect } from '../../components/ui/TagInput';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';

const blank: Omit<Webhook, 'id'> = { url: '', events: ['call.ended'], secret: '', enabled: true };

function randomSecret() {
  const a = new Uint8Array(24);
  crypto.getRandomValues(a);
  return Array.from(a, (b) => b.toString(16).padStart(2, '0')).join('');
}

export function WebhooksTab() {
  const crud = useCrud<Webhook>('/webhooks');
  const [edit, setEdit] = useState<(Omit<Webhook, 'id'> & { id?: number }) | null>(null);
  const [busy, setBusy] = useState(false);

  const save = async () => {
    if (!edit) return;
    setBusy(true);
    try { await crud.save(edit); toast.success('Webhook saved'); setEdit(null); } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  const remove = async (w: Webhook) => {
    if (!(await confirm({ title: 'Delete webhook?', message: `Stop sending events to ${w.url}?`, danger: true, confirmLabel: 'Delete' }))) return;
    try { await crud.remove(w.id); toast.success('Webhook deleted'); } catch (e) { toast.error(e); }
  };
  const all = edit?.events.includes('*');
  const opts = EVENT_TYPES.map((e) => ({ value: e, label: e }));

  return (
    <Card title="Webhooks" icon={<WebhookIcon />} flush actions={<button className="btn sm primary" onClick={() => setEdit({ ...blank, secret: randomSecret() })}><Plus />Add webhook</button>}>
      {crud.error ? <div style={{ padding: 16 }}><ErrorBox error={crud.error} onRetry={crud.reload} /></div>
        : crud.loading && !crud.data ? <SkeletonRows rows={3} />
        : crud.items.length === 0 ? <Empty title="No webhooks" icon={<WebhookIcon />}>Events are POSTed as JSON, signed with HMAC-SHA256 in <code>X-Callcenter-Signature</code>.</Empty>
        : (
          <div className="list">
            {crud.items.map((w) => (
              <div key={w.id} className="list-item">
                <div className="grow" style={{ minWidth: 0 }}>
                  <div className="strong ellipsis mono">{w.url}</div>
                  <div className="chips" style={{ marginTop: 4 }}>{(w.events ?? []).map((e) => <span key={e} className="chip">{e === '*' ? 'all events' : e}</span>)}</div>
                </div>
                <Toggle checked={w.enabled} onChange={(v) => crud.save({ ...w, enabled: v }).catch(toast.error)} />
                <button className="btn sm ghost icon" aria-label="Edit" onClick={() => setEdit({ ...w })}><Pencil /></button>
                <button className="btn sm ghost icon" aria-label="Delete" onClick={() => remove(w)}><Trash2 /></button>
              </div>
            ))}
          </div>
        )}
      <Dialog open={!!edit} onClose={() => setEdit(null)} title={edit?.id ? 'Edit webhook' : 'New webhook'}
        footer={<><button className="btn" onClick={() => setEdit(null)}>Cancel</button><button className="btn primary" disabled={busy || !edit?.url.match(/^https?:\/\//) || !edit?.events.length} onClick={save}>Save</button></>}>
        {edit && (
          <>
            <Field label="URL"><input className="input" placeholder="https://example.com/hooks/callcenter" value={edit.url} onChange={(e) => setEdit({ ...edit, url: e.target.value })} data-autofocus /></Field>
            <div className="field">
              <span className="field-label">Events</span>
              <Toggle checked={!!all} onChange={(v) => setEdit({ ...edit, events: v ? ['*'] : [] })} label="All events (*)" />
              {!all && <ChipSelect options={opts} value={edit.events} onChange={(v) => setEdit({ ...edit, events: v })} />}
            </div>
            <Field label="Signing secret" hint="Used to compute the HMAC signature header">
              <div className="row"><input className="input mono" value={edit.secret} onChange={(e) => setEdit({ ...edit, secret: e.target.value })} />
                <button className="btn sm" type="button" onClick={() => setEdit({ ...edit, secret: randomSecret() })}>Generate</button></div>
            </Field>
            <Toggle checked={edit.enabled} onChange={(v) => setEdit({ ...edit, enabled: v })} label="Enabled" />
          </>
        )}
      </Dialog>
    </Card>
  );
}
