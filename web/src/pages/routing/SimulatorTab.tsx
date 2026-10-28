import { useState, type FormEvent } from 'react';
import { FlaskConical, Play } from 'lucide-react';
import { post } from '../../api/client';
import type { RoutingDecision } from '../../api/types_routing';
import { useLookup } from '../../state/lookups';
import { Card, Empty, Field, Toggle } from '../../components/ui/primitives';
import { toast } from '../../components/ui/Toast';
import { DecisionView } from './DecisionView';

const EXAMPLES = ['I want a refund', 'I was charged twice, I will call my lawyer', 'I cannot log in to my account', 'می‌خواهم پولم را پس بگیرم'];

/** Shows where a request would be routed right now (approved rules only), without touching any call. */
export function SimulatorTab() {
  const [text, setText] = useState('');
  const [topic, setTopic] = useState('');
  const [language, setLanguage] = useState('');
  const [vip, setVip] = useState(false);
  const [busy, setBusy] = useState(false);
  const [decision, setDecision] = useState<RoutingDecision | null>(null);
  const topics = useLookup('topics');
  const run = async (e?: FormEvent) => {
    e?.preventDefault();
    setBusy(true);
    try { setDecision(await post<RoutingDecision>('/routing/simulate', { text, topic: topic || undefined, language: language || undefined, vip })); }
    catch (err) { toast.error(err); } finally { setBusy(false); }
  };
  return (
    <div className="grid-2" style={{ alignItems: 'start' }}>
      <Card title="Simulate a request" icon={<FlaskConical />}>
        <form className="col gap-lg" onSubmit={run}>
          <Field label="What the caller says">
            <textarea className="textarea" dir="auto" value={text} onChange={(e) => setText(e.target.value)} placeholder="e.g. I want a refund for my last order" />
          </Field>
          <div className="chips">{EXAMPLES.map((x) => <button key={x} type="button" className="btn sm ghost" dir="auto" onClick={() => setText(x)}>{x}</button>)}</div>
          <div className="form-grid">
            <Field label="Topic (optional)">
              <select className="select" value={topic} onChange={(e) => setTopic(e.target.value)}>
                <option value="">Detect from text</option>
                {topics.map((t) => <option key={t.id} value={t.name}>{t.name}</option>)}
              </select>
            </Field>
            <Field label="Language (optional)"><input className="input" placeholder="e.g. en, fa" value={language} onChange={(e) => setLanguage(e.target.value)} /></Field>
          </div>
          <Toggle checked={vip} onChange={setVip} label="VIP caller" />
          <div><button className="btn primary" disabled={busy || !(text.trim() || topic)}><Play />Simulate</button></div>
        </form>
      </Card>
      <Card title="Decision">
        {decision ? <DecisionView d={decision} /> : <Empty title="No simulation yet">Enter a request to see where it would go and why.</Empty>}
      </Card>
    </div>
  );
}
