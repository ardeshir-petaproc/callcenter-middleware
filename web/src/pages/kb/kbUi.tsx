import { Bot, Lock } from 'lucide-react';
import type { KnowledgeSource } from '../../api/types_work';

export function isExpired(s: KnowledgeSource): boolean {
  if (!s.valid_until) return false;
  const d = Date.parse(`${s.valid_until}T23:59:59`);
  return !Number.isNaN(d) && d < Date.now();
}

/** Status / expiry / visibility / AI badges for a knowledge source. */
export function KBBadges({ s }: { s: KnowledgeSource }) {
  return (
    <span className="row wrap" style={{ gap: 4 }}>
      {s.status === 'draft' && <span className="badge warn">Draft</span>}
      {s.status === 'archived' && <span className="badge">Archived</span>}
      {isExpired(s) && <span className="badge danger" title={`Valid until ${s.valid_until}`}>Expired</span>}
      {s.visibility === 'restricted' && <span className="badge violet"><Lock size={11} />Restricted</span>}
      {s.ai_enabled && <span className="badge info" title="The AI may use this source"><Bot size={11} />AI</span>}
    </span>
  );
}

export function fmtBytes(n: number): string {
  if (!n) return '—';
  if (n < 1024) return `${n} B`;
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1024 / 1024).toFixed(1)} MB`;
}

export const KB_ACCEPT = '.txt,.md,.csv,.json,.html,.htm,.docx,.pptx,.xlsx,.pdf';
