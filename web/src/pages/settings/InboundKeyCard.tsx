import { useState } from 'react';
import { KeyRound, MessagesSquare } from 'lucide-react';
import { post } from '../../api/client';
import { Card, Field } from '../../components/ui/primitives';
import { CopyField } from '../../components/ui/CopyField';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';

interface InboundKey { key: string; header: string; endpoint: string }

/** Joow chat messages → tickets: issue the key the messaging integration authenticates with. */
export function InboundKeyCard() {
  const [res, setRes] = useState<InboundKey | null>(null);
  const [busy, setBusy] = useState(false);
  const generate = async () => {
    if (!(await confirm({ title: 'Generate a new inbound key?', message: 'Any previously issued key stops working immediately.', confirmLabel: 'Generate key' }))) return;
    setBusy(true);
    try { setRes(await post<InboundKey>('/integrations/inbound-key')); } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  return (
    <Card title="Chat messages (Joow)" icon={<MessagesSquare />} actions={<button className="btn sm" disabled={busy} onClick={generate}><KeyRound />Generate inbound key</button>}>
      <p className="small muted">Customer chat messages delivered by Joow (or another channel) to the inbound endpoint become tickets with source “message”.</p>
      {res && (
        <div className="col gap-lg" style={{ marginTop: 12 }}>
          <div className="alert warn"><KeyRound /><div>Copy the key now — it is stored hashed and <strong>will not be shown again</strong>.</div></div>
          <Field label="Key"><CopyField value={res.key} label="Key" /></Field>
          <Field label="Header"><CopyField value={res.header} label="Header" /></Field>
          <Field label="Endpoint (POST)"><CopyField value={res.endpoint} label="Endpoint" /></Field>
        </div>
      )}
    </Card>
  );
}
