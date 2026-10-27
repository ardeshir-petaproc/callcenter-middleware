import { useState } from 'react';
import { Delete, PhoneOutgoing } from 'lucide-react';
import { post } from '../../api/client';
import { Card } from '../../components/ui/primitives';
import { Dialpad } from '../../components/ui/Dialpad';
import { toast } from '../../components/ui/Toast';

/** Click-to-call: the server rings the agent's own phone first, then the number. */
export function DialerCard() {
  const [number, setNumber] = useState('');
  const [busy, setBusy] = useState(false);
  const dial = async () => {
    const n = number.trim();
    if (!n) return;
    setBusy(true);
    try {
      await post<{ call_id: string }>('/me/dial', { number: n });
      toast.success(`Calling ${n} — answer your phone to connect`);
      setNumber('');
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  return (
    <Card title="Click-to-call" icon={<PhoneOutgoing />}>
      <form className="col gap-lg" onSubmit={(e) => { e.preventDefault(); void dial(); }}>
        <div className="row">
          <input className="input" style={{ fontSize: 18, height: 42, fontVariantNumeric: 'tabular-nums' }} inputMode="tel"
            placeholder="Phone number" value={number} onChange={(e) => setNumber(e.target.value.replace(/[^\d+*#\s()-]/g, ''))} aria-label="Phone number" />
          <button type="button" className="btn ghost icon" aria-label="Backspace" onClick={() => setNumber((n) => n.slice(0, -1))}><Delete /></button>
        </div>
        <Dialpad onPress={(k) => setNumber((n) => n + k)} />
        <button className="btn success lg" type="submit" disabled={busy || !number.trim()}><PhoneOutgoing />Call</button>
      </form>
    </Card>
  );
}
