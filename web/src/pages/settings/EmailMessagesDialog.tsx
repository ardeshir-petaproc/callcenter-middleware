import { Link } from 'react-router-dom';
import { get } from '../../api/client';
import type { EmailConnector, EmailMessage } from '../../api/types_work';
import { useApi } from '../../lib/hooks';
import { fmtDateTime } from '../../lib/format';
import { Dialog } from '../../components/ui/Dialog';
import { Empty, ErrorBox, SkeletonRows } from '../../components/ui/primitives';

/** Imported messages of one connector (headers only) with links to the ticket / knowledge they created. */
export function EmailMessagesDialog({ connector, onClose }: { connector: EmailConnector | null; onClose: () => void }) {
  const id = connector?.id ?? 0;
  const { data, loading, error, reload } = useApi((s) => get<EmailMessage[]>(`/email-connectors/${id}/messages`, { limit: 200 }, s), [id], !!id);
  return (
    <Dialog open={!!connector} onClose={onClose} title={`Imported from ${connector?.name ?? ''}`} size="wide">
      {error ? <ErrorBox error={error} onRetry={reload} />
        : loading && !data ? <SkeletonRows rows={4} />
        : !data?.length ? <Empty title="Nothing imported yet" />
        : (
          <div className="table-wrap">
            <table className="table">
              <thead><tr><th>Date</th><th>From</th><th>Subject</th><th>Created</th></tr></thead>
              <tbody>
                {data.map((m) => (
                  <tr key={m.id}>
                    <td className="tnum">{fmtDateTime(m.date)}</td>
                    <td><span dir="auto">{m.from_name || m.from_addr}</span>{m.from_name && <div className="sub">{m.from_addr}</div>}</td>
                    <td dir="auto">{m.subject || '(no subject)'}</td>
                    <td>
                      {m.ticket_id > 0 && <Link to={`/tickets/${m.ticket_id}`} onClick={onClose}>Ticket</Link>}
                      {m.kb_source_id > 0 && <Link to={`/kb?source=${m.kb_source_id}`} onClick={onClose}>Knowledge</Link>}
                      {!m.ticket_id && !m.kb_source_id && <span className="muted">—</span>}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
    </Dialog>
  );
}
