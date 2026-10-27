import { Bar, BarChart, CartesianGrid, ResponsiveContainer, Tooltip, XAxis, YAxis, type TooltipProps } from 'recharts';
import type { Bucket, LabelCount } from '../../api/types';
import { fmtNum, humanize } from '../../lib/format';
import { Empty } from '../../components/ui/primitives';

const axis = { fontSize: 11, fill: 'var(--text-3)' };

function Tip({ active, payload, label }: TooltipProps<number, string>) {
  if (!active || !payload?.length) return null;
  return (
    <div className="chart-tooltip">
      <div className="strong" style={{ marginBottom: 4 }}>{label}</div>
      {payload.map((p) => (
        <div key={p.dataKey as string} className="row"><i style={{ background: p.color }} />{p.name}: <b className="num">{fmtNum(p.value as number)}</b></div>
      ))}
    </div>
  );
}

function hourLabel(key: string): string {
  const m = key.match(/(\d{1,2})(?::\d{2})?$/) ?? key.match(/T(\d{2})/);
  return m ? `${m[1].padStart(2, '0')}:00` : key;
}
function dayLabel(key: string): string {
  const d = new Date(`${key.slice(0, 10)}T00:00:00`);
  return Number.isNaN(d.getTime()) ? key : d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}

/** Fill empty hours (00-23) or days (from..to) so the time axis is continuous. */
function fillBuckets(data: Bucket[], by: 'hour' | 'day', from?: string, to?: string): Bucket[] {
  const zero = (key: string): Bucket => ({ key, total: 0, answered: 0, abandoned: 0 });
  if (by === 'hour') {
    const m = new Map(data.map((b) => [Number(hourLabel(b.key).slice(0, 2)), b]));
    if ([...m.keys()].some(Number.isNaN)) return data;
    return Array.from({ length: 24 }, (_, h) => m.get(h) ?? zero(String(h).padStart(2, '0')));
  }
  if (!from || !to) return data;
  const m = new Map(data.map((b) => [b.key.slice(0, 10), b]));
  const out: Bucket[] = [];
  const d = new Date(`${from}T00:00:00`);
  const end = new Date(`${to}T00:00:00`);
  for (let i = 0; d <= end && i < 400; i++, d.setDate(d.getDate() + 1)) {
    const k = `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
    out.push(m.get(k) ?? zero(k));
  }
  return out;
}

/** Answered vs abandoned per bucket (two series → legend + tooltip). */
export function VolumeChart({ data, by, from, to }: { data: Bucket[]; by: 'hour' | 'day'; from?: string; to?: string }) {
  if (!data.length || data.every((b) => !b.total)) return <Empty title="No calls in this period" />;
  const rows = fillBuckets(data, by, from, to).map((b) => ({ ...b, label: by === 'hour' ? hourLabel(b.key) : dayLabel(b.key) }));
  return (
    <>
      <div className="chart-legend" style={{ marginBottom: 8 }}>
        <span><i style={{ background: 'var(--series-1)' }} />Answered</span>
        <span><i style={{ background: 'var(--series-2)' }} />Abandoned</span>
      </div>
      <div className="chart-box">
        <ResponsiveContainer>
          <BarChart data={rows} barGap={2} margin={{ top: 4, right: 4, bottom: 0, left: -18 }}>
            <CartesianGrid vertical={false} stroke="var(--chart-grid)" />
            <XAxis dataKey="label" tick={axis} tickLine={false} axisLine={{ stroke: 'var(--border)' }} interval="preserveStartEnd" minTickGap={8} />
            <YAxis allowDecimals={false} tick={axis} tickLine={false} axisLine={false} />
            <Tooltip content={<Tip />} cursor={{ fill: 'var(--surface-3)', opacity: 0.6 }} />
            <Bar dataKey="answered" name="Answered" fill="var(--series-1)" radius={[4, 4, 0, 0]} maxBarSize={22} />
            <Bar dataKey="abandoned" name="Abandoned" fill="var(--series-2)" radius={[4, 4, 0, 0]} maxBarSize={22} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </>
  );
}

/** Single-series horizontal bar breakdown with direct value labels. */
export function BreakdownChart({ data }: { data: LabelCount[] }) {
  const rows = [...data].filter((d) => d.count > 0).sort((a, b) => b.count - a.count).slice(0, 10)
    .map((d) => ({ ...d, name: d.label ? humanize(d.label) : 'None' }));
  if (!rows.length) return <Empty title="No data" />;
  const total = rows.reduce((n, r) => n + r.count, 0);
  return (
    <div className="col" style={{ gap: 8 }}>
      {rows.map((r) => (
        <div key={r.label} title={`${r.name}: ${fmtNum(r.count)}`}>
          <div className="row between small"><span className="ellipsis">{r.name}</span><span className="num muted">{fmtNum(r.count)} · {Math.round((r.count / total) * 100)}%</span></div>
          <div className="bar" style={{ height: 8, marginTop: 3 }}><span style={{ width: `${(r.count / rows[0].count) * 100}%`, background: 'var(--series-1)', borderRadius: 4 }} /></div>
        </div>
      ))}
    </div>
  );
}
