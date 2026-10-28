import { useState } from 'react';
import { Lock, MessageSquare, Send } from 'lucide-react';
import { post } from '../../api/client';
import type { TicketComment } from '../../api/types_work';
import { fmtDateTime } from '../../lib/format';
import { Card, Empty, Segmented } from '../../components/ui/primitives';
import { toast } from '../../components/ui/Toast';

type Kind = 'reply' | 'note';

/** Comment thread: customer-visible replies and staff-only internal notes, plus a composer. */
export function TicketThread({ ticketId, comments, onPosted }: { ticketId: number; comments: TicketComment[]; onPosted: () => void }) {
  const [kind, setKind] = useState<Kind>('reply');
  const [body, setBody] = useState('');
  const [busy, setBusy] = useState(false);
  const send = async () => {
    setBusy(true);
    try {
      await post(`/tickets/${ticketId}/comments`, { body: body.trim(), internal: kind === 'note' });
      setBody('');
      onPosted();
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  return (
    <Card title={`Conversation (${comments.length})`} icon={<MessageSquare />}>
      <div className="thread">
        {comments.length === 0 && <Empty title="No replies yet" />}
        {comments.map((c) => (
          <div key={c.id} className={`comment ${c.internal ? 'internal' : ''}`}>
            <div className="comment-head">
              <strong dir="auto">{c.author_name || 'System'}</strong>
              {c.internal ? <span className="badge warn"><Lock size={11} />Internal note</span> : <span className="badge info">Reply to customer</span>}
              <span className="small muted">{fmtDateTime(c.created_at)}</span>
            </div>
            <div className="pre" dir="auto">{c.body}</div>
          </div>
        ))}
      </div>
      <div className={`composer ${kind === 'note' ? 'internal' : ''}`}>
        <Segmented<Kind> label="Comment type" value={kind} onChange={setKind}
          options={[{ value: 'reply', label: 'Reply to customer' }, { value: 'note', label: 'Internal note' }]} />
        <textarea className="textarea" dir="auto" value={body} onChange={(e) => setBody(e.target.value)}
          placeholder={kind === 'note' ? 'Only staff can see internal notes…' : 'The customer can see this reply…'}
          onKeyDown={(e) => { if (e.key === 'Enter' && (e.ctrlKey || e.metaKey) && body.trim()) void send(); }} />
        <div className="row between">
          <span className="small muted">{kind === 'note' ? 'Visible to staff only.' : 'Visible to the customer.'} Ctrl+Enter to send.</span>
          <button className="btn primary" disabled={busy || !body.trim()} onClick={send}>{kind === 'note' ? <Lock /> : <Send />}{kind === 'note' ? 'Add note' : 'Send reply'}</button>
        </div>
      </div>
    </Card>
  );
}
