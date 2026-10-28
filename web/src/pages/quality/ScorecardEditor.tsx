import { useEffect, useState } from 'react';
import { Plus, Trash2 } from 'lucide-react';
import type { QACriterion, QAScorecard } from '../../api/types_routing';
import { Dialog } from '../../components/ui/Dialog';
import { Field, Toggle } from '../../components/ui/primitives';

type Draft = Omit<QAScorecard, 'id'> & { id?: number };
const blank: Draft = {
  name: '', description: '', pass_score: 80, enabled: true,
  criteria: [{ key: '', label: 'Greeting and identification', weight: 1, max_score: 5, critical: false }],
};

/** Scorecard editor: criteria with label, weight, max score and the critical (auto-fail) flag. */
export function ScorecardEditor({ open, card, onClose, onSave }: { open: boolean; card: QAScorecard | null; onClose: () => void; onSave: (c: Draft) => Promise<void> }) {
  const [c, setC] = useState<Draft>(blank);
  const [busy, setBusy] = useState(false);
  useEffect(() => { if (open) setC(card ? { ...card, criteria: card.criteria ?? [] } : { ...blank, criteria: blank.criteria.map((x) => ({ ...x })) }); }, [open, card]);
  const setCrit = (i: number, p: Partial<QACriterion>) => setC((x) => ({ ...x, criteria: x.criteria.map((k, j) => (j === i ? { ...k, ...p } : k)) }));
  const valid = c.name.trim() && c.criteria.length > 0 && c.criteria.every((k) => k.label.trim());
  const save = async () => { setBusy(true); try { await onSave(c); onClose(); } catch { /* reported by onSave */ } finally { setBusy(false); } };
  const totalWeight = c.criteria.reduce((s, k) => s + Math.max(k.weight, 1), 0);

  return (
    <Dialog open={open} onClose={onClose} size="wide" title={card ? `Edit “${card.name}”` : 'New scorecard'}
      footer={<><button className="btn" onClick={onClose}>Cancel</button><button className="btn primary" disabled={busy || !valid} onClick={save}>Save scorecard</button></>}>
      <div className="form-grid">
        <Field label="Name"><input className="input" dir="auto" value={c.name} onChange={(e) => setC({ ...c, name: e.target.value })} data-autofocus /></Field>
        <Field label="Pass score (%)"><input className="input" type="number" min={0} max={100} value={c.pass_score} onChange={(e) => setC({ ...c, pass_score: Number(e.target.value) })} /></Field>
        <div className="field" style={{ justifyContent: 'flex-end' }}><Toggle checked={c.enabled} onChange={(v) => setC({ ...c, enabled: v })} label="Enabled" /></div>
        <Field label="Description" className="span-all"><input className="input" dir="auto" value={c.description} onChange={(e) => setC({ ...c, description: e.target.value })} /></Field>
      </div>
      <div className="form-section">
        <h3>Criteria</h3>
        <div className="table-wrap">
          <table className="table">
            <thead><tr><th>Label</th><th className="num">Weight</th><th className="num">Max score</th><th>Critical</th><th /></tr></thead>
            <tbody>
              {c.criteria.map((k, i) => (
                <tr key={i}>
                  <td><input className="input sm" dir="auto" value={k.label} onChange={(e) => setCrit(i, { label: e.target.value })} aria-label="Criterion label" /></td>
                  <td className="num"><input className="input sm" type="number" min={1} style={{ width: 70 }} value={k.weight} onChange={(e) => setCrit(i, { weight: Number(e.target.value) })} aria-label="Weight" />
                    <div className="sub">{Math.round((Math.max(k.weight, 1) / totalWeight) * 100)}%</div></td>
                  <td className="num"><input className="input sm" type="number" min={1} style={{ width: 70 }} value={k.max_score} onChange={(e) => setCrit(i, { max_score: Number(e.target.value) })} aria-label="Max score" /></td>
                  <td><label className="checkbox small" title="Scoring 0 fails the whole evaluation"><input type="checkbox" checked={k.critical} onChange={(e) => setCrit(i, { critical: e.target.checked })} />Auto-fail at 0</label></td>
                  <td className="actions"><button className="btn sm ghost icon" aria-label="Remove criterion" disabled={c.criteria.length <= 1} onClick={() => setC({ ...c, criteria: c.criteria.filter((_, j) => j !== i) })}><Trash2 /></button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        <div><button type="button" className="btn sm" onClick={() => setC({ ...c, criteria: [...c.criteria, { key: '', label: '', weight: 1, max_score: 5, critical: false }] })}><Plus />Add criterion</button></div>
      </div>
    </Dialog>
  );
}
