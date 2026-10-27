import type { CampaignStats, CampaignStatus, LeadStatus } from '../../api/types';
import { humanize } from '../../lib/format';

const statusTone: Record<CampaignStatus, string> = { draft: '', running: 'ok', paused: 'warn', completed: 'info' };
export function CampaignStatusBadge({ status }: { status: CampaignStatus }) {
  return <span className={`badge ${statusTone[status] ?? ''} ${status === 'running' ? 'pulse' : ''}`}><span className="dot" />{humanize(status)}</span>;
}

const leadTone: Record<LeadStatus, string> = { pending: '', dialing: 'info', answered: 'ok', failed: 'danger', done: 'violet' };
export function LeadStatusBadge({ status }: { status: LeadStatus }) {
  return <span className={`badge ${leadTone[status] ?? ''}`}>{humanize(status)}</span>;
}

/** Progress = finished leads (answered + failed + done) over total, split by colour. */
export function CampaignProgress({ stats, showLabel }: { stats?: CampaignStats; showLabel?: boolean }) {
  const l = stats?.leads ?? {};
  const total = l.total ?? 0;
  const answered = (l.answered ?? 0) + (l.done ?? 0);
  const failed = l.failed ?? 0;
  const pct = total ? Math.round(((answered + failed) / total) * 100) : 0;
  return (
    <div className="col" style={{ gap: 3 }}>
      <div className="bar" role="progressbar" aria-valuenow={pct} aria-valuemin={0} aria-valuemax={100} aria-label="Campaign progress">
        <span style={{ width: `${total ? (answered / total) * 100 : 0}%`, background: 'var(--ok-solid)' }} />
        <span style={{ width: `${total ? (failed / total) * 100 : 0}%`, background: 'var(--danger-solid)' }} />
      </div>
      {showLabel && <div className="small muted num">{pct}% · {answered + failed} of {total} leads</div>}
    </div>
  );
}

/** Minimal CSV parser (quotes, commas, CRLF) → rows of cells. */
export function parseCsv(text: string): string[][] {
  const rows: string[][] = [];
  let row: string[] = [];
  let cell = '';
  let quoted = false;
  for (let i = 0; i < text.length; i++) {
    const ch = text[i];
    if (quoted) {
      if (ch === '"' && text[i + 1] === '"') { cell += '"'; i++; }
      else if (ch === '"') quoted = false;
      else cell += ch;
    } else if (ch === '"') quoted = true;
    else if (ch === ',' || ch === ';' || ch === '\t') { row.push(cell.trim()); cell = ''; }
    else if (ch === '\n' || ch === '\r') {
      if (ch === '\r' && text[i + 1] === '\n') i++;
      row.push(cell.trim()); cell = '';
      if (row.some((c) => c)) rows.push(row);
      row = [];
    } else cell += ch;
  }
  row.push(cell.trim());
  if (row.some((c) => c)) rows.push(row);
  return rows;
}
