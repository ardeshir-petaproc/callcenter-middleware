import { useState, type FormEvent } from 'react';
import { Bot, Search } from 'lucide-react';
import { get, post } from '../../api/client';
import type { KBHit } from '../../api/types_work';
import { useLookup } from '../../state/lookups';
import { Card, Empty, Segmented } from '../../components/ui/primitives';
import { toast } from '../../components/ui/Toast';

type Mode = 'search' | 'ai';

/** Full-text search over readable sources; managers can preview what an AI agent would retrieve. */
export function KBSearchPanel({ canManage, onOpen }: { canManage: boolean; onOpen: (sourceId: number) => void }) {
  const [mode, setMode] = useState<Mode>('search');
  const [q, setQ] = useState('');
  const [profile, setProfile] = useState(0);
  const [hits, setHits] = useState<KBHit[] | null>(null);
  const [busy, setBusy] = useState(false);
  const profiles = useLookup('aiProfiles');

  const run = async (e?: FormEvent) => {
    e?.preventDefault();
    if (!q.trim()) return;
    setBusy(true);
    try {
      setHits(mode === 'ai'
        ? await post<KBHit[]>('/kb/search/ai', { q: q.trim(), profile_id: profile || undefined })
        : await get<KBHit[]>('/kb/search', { q: q.trim(), limit: 10 }));
    } catch (err) { toast.error(err); } finally { setBusy(false); }
  };

  return (
    <Card flush>
      <form className="row wrap" style={{ gap: 8, padding: '12px 16px' }} onSubmit={run}>
        {canManage && <Segmented<Mode> label="Search mode" value={mode} onChange={(m) => { setMode(m); setHits(null); }}
          options={[{ value: 'search', label: 'Search' }, { value: 'ai', label: 'AI preview' }]} />}
        <div className="input-icon" style={{ flex: '1 1 260px' }}>
          <Search /><input className="input" dir="auto" placeholder={mode === 'ai' ? 'What would the AI find for… (e.g. refund policy)' : 'Search inside documents…'} value={q} onChange={(e) => setQ(e.target.value)} aria-label="Search knowledge" />
        </div>
        {mode === 'ai' && (
          <select className="select" style={{ width: 200 }} value={profile || ''} onChange={(e) => setProfile(Number(e.target.value) || 0)} aria-label="AI profile">
            <option value="">Any AI agent (no scope)</option>
            {profiles.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
          </select>
        )}
        <button className="btn primary" disabled={busy || !q.trim()}>{mode === 'ai' ? <Bot /> : <Search />}{mode === 'ai' ? 'Preview' : 'Search'}</button>
        {hits && <button type="button" className="btn ghost" onClick={() => setHits(null)}>Clear</button>}
      </form>
      {hits && (
        <div style={{ borderTop: '1px solid var(--border)' }}>
          {mode === 'ai' && <div className="small muted" style={{ padding: '8px 16px 0' }}>Only active, unexpired, AI-enabled sources within the agent's knowledge scope are retrieved.</div>}
          {hits.length === 0 ? <Empty title="No matches" /> : (
            <div className="list">
              {hits.map((h, i) => (
                <button key={`${h.source_id}-${h.chunk ?? i}`} type="button" className="list-item clickable plain-button" onClick={() => onOpen(h.source_id)}>
                  <div className="grow" style={{ textAlign: 'start' }}>
                    <div className="row wrap"><span className="strong" dir="auto">{h.title}</span>{h.category && <span className="chip" dir="auto">{h.category}</span>}<span className="small muted">score {h.score.toFixed(2)}</span></div>
                    <div className="small snippet" dir="auto">{h.text || h.snippet}</div>
                  </div>
                </button>
              ))}
            </div>
          )}
        </div>
      )}
    </Card>
  );
}
