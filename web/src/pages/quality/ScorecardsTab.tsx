import { useState } from 'react';
import { ClipboardList, Pencil, Plus, Trash2 } from 'lucide-react';
import type { QAScorecard } from '../../api/types_routing';
import { useCrud } from '../../lib/crud';
import { Card, Empty, ErrorBox, SkeletonRows } from '../../components/ui/primitives';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';
import { RevisionsButton } from '../../components/RevisionsButton';
import { ScorecardEditor } from './ScorecardEditor';

export function ScorecardsTab() {
  const crud = useCrud<QAScorecard>('/qa/scorecards');
  const [edit, setEdit] = useState<QAScorecard | null | 'new'>(null);
  const remove = async (c: QAScorecard) => {
    if (!(await confirm({ title: 'Delete scorecard?', message: `Delete “${c.name}”? Existing evaluations keep their scores.`, danger: true, confirmLabel: 'Delete' }))) return;
    try { await crud.remove(c.id); } catch (e) { toast.error(e); }
  };
  return (
    <Card title="Scorecards" icon={<ClipboardList />} flush actions={<button className="btn sm primary" onClick={() => setEdit('new')}><Plus />New scorecard</button>}>
      {crud.error ? <div style={{ padding: 16 }}><ErrorBox error={crud.error} onRetry={crud.reload} /></div>
        : crud.loading && !crud.data ? <SkeletonRows rows={3} />
        : crud.items.length === 0 ? <Empty title="No scorecards" icon={<ClipboardList />}>Create a scorecard to start evaluating calls.</Empty>
        : (
          <div className="list">
            {crud.items.map((c) => (
              <div key={c.id} className="list-item" style={{ alignItems: 'flex-start' }}>
                <div className="grow">
                  <div className="row wrap"><span className="strong" dir="auto">{c.name}</span><span className="badge info">pass ≥ {c.pass_score}%</span>{!c.enabled && <span className="badge">Disabled</span>}</div>
                  {c.description && <div className="small muted" dir="auto">{c.description}</div>}
                  <div className="chips" style={{ marginTop: 4 }}>{(c.criteria ?? []).map((k) => <span key={k.key} className="chip" dir="auto">{k.label} ×{k.weight}{k.critical ? ' ⚑' : ''}</span>)}</div>
                </div>
                <RevisionsButton entity="qa_scorecard" id={c.id} label="" />
                <button className="btn sm ghost icon" aria-label="Edit" onClick={() => setEdit(c)}><Pencil /></button>
                <button className="btn sm ghost icon" aria-label="Delete" onClick={() => remove(c)}><Trash2 /></button>
              </div>
            ))}
          </div>
        )}
      <ScorecardEditor open={edit !== null} card={edit === 'new' ? null : edit} onClose={() => setEdit(null)}
        onSave={async (c) => { try { await crud.save(c as Partial<QAScorecard>); toast.success('Scorecard saved'); } catch (e) { toast.error(e); throw e; } }} />
    </Card>
  );
}
