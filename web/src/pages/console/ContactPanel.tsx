import { Link } from 'react-router-dom';
import { Building2, Mail, Star, UserRound } from 'lucide-react';
import type { Call, Contact } from '../../api/types';
import { get } from '../../api/client';
import { useApi } from '../../lib/hooks';
import { fmtDateTime, fmtDuration } from '../../lib/format';
import { OutcomeBadge } from '../../components/ui/Badges';
import { Skeleton } from '../../components/ui/primitives';

interface Lookup { contact: Contact | null; recent_calls: Call[] }

/** CRM record + recent call history for a phone number (via /contacts/lookup). */
export function ContactPanel({ phone, excludeCallId }: { phone: string; excludeCallId?: string }) {
  const { data, loading } = useApi((signal) => get<Lookup>('/contacts/lookup', { phone }, signal), [phone], !!phone);
  if (!phone) return null;
  if (loading) return <div className="col"><Skeleton h={18} w="60%" /><Skeleton h={14} w="40%" /></div>;
  const c = data?.contact;
  const recent = (data?.recent_calls ?? []).filter((x) => x.id !== excludeCallId).slice(0, 5);
  return (
    <div className="col gap-lg">
      {c ? (
        <div className="col" style={{ gap: 4 }}>
          <div className="row">
            <UserRound size={16} className="muted" />
            <span className="strong">{c.name || 'Unnamed contact'}</span>
            {c.vip && <span className="badge warn"><Star size={12} />VIP</span>}
          </div>
          {c.company && <div className="row small"><Building2 size={14} className="muted" />{c.company}</div>}
          {c.email && <div className="row small"><Mail size={14} className="muted" />{c.email}</div>}
          {c.tags?.length > 0 && <div className="chips">{c.tags.map((t) => <span key={t} className="chip">{t}</span>)}</div>}
          {c.notes && <div className="small pre" style={{ background: 'var(--surface-2)', padding: 8, borderRadius: 6 }}>{c.notes}</div>}
        </div>
      ) : (
        <div className="small muted">No contact record for {phone}. <Link to={`/contacts?new=${encodeURIComponent(phone)}`}>Create one</Link></div>
      )}
      <div>
        <div className="field-label" style={{ marginBottom: 6 }}>Recent calls</div>
        {recent.length === 0 ? <div className="small muted">First time caller</div> : (
          <div className="col" style={{ gap: 4 }}>
            {recent.map((r) => (
              <Link key={r.id} to={`/calls/${encodeURIComponent(r.id)}`} className="row between small" style={{ color: 'inherit' }}>
                <span>{fmtDateTime(r.started_at)}</span>
                <span className="row" style={{ gap: 6 }}>
                  {r.disposition && <span className="muted">{r.disposition}</span>}
                  <span className="num muted">{fmtDuration(r.talk_sec)}</span>
                  <OutcomeBadge outcome={r.outcome} />
                </span>
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
