import { toLocalDateInput } from '../../lib/format';

export interface Range { from: string; to: string; preset: Preset }
export type Preset = 'today' | 'yesterday' | '7d' | '30d' | 'custom';

export function presetRange(p: Preset): Range {
  const today = new Date();
  const d = (offset: number) => { const x = new Date(today); x.setDate(x.getDate() - offset); return toLocalDateInput(x); };
  switch (p) {
    case 'yesterday': return { from: d(1), to: d(1), preset: p };
    case '7d': return { from: d(6), to: d(0), preset: p };
    case '30d': return { from: d(29), to: d(0), preset: p };
    default: return { from: d(0), to: d(0), preset: 'today' };
  }
}

const PRESETS: { value: Preset; label: string }[] = [
  { value: 'today', label: 'Today' }, { value: 'yesterday', label: 'Yesterday' }, { value: '7d', label: 'Last 7 days' }, { value: '30d', label: 'Last 30 days' },
];

export function DateRangePicker({ value, onChange }: { value: Range; onChange: (r: Range) => void }) {
  return (
    <div className="row wrap" style={{ gap: 8 }}>
      <div className="segmented" role="group" aria-label="Date presets">
        {PRESETS.map((p) => (
          <button key={p.value} className={value.preset === p.value ? 'active' : ''} aria-pressed={value.preset === p.value} onClick={() => onChange(presetRange(p.value))}>{p.label}</button>
        ))}
      </div>
      <input className="input sm" type="date" style={{ width: 140 }} value={value.from} max={value.to} aria-label="From"
        onChange={(e) => e.target.value && onChange({ ...value, from: e.target.value, preset: 'custom' })} />
      <span className="muted small">to</span>
      <input className="input sm" type="date" style={{ width: 140 }} value={value.to} min={value.from} aria-label="To"
        onChange={(e) => e.target.value && onChange({ ...value, to: e.target.value, preset: 'custom' })} />
    </div>
  );
}
