import { Copy, Plus, Trash2 } from 'lucide-react';
import type { Schedule, WeeklyRange } from '../../api/types';
import { Field } from '../../components/ui/primitives';

const DAYS = ['Sunday', 'Monday', 'Tuesday', 'Wednesday', 'Thursday', 'Friday', 'Saturday'];
const ORDER = [1, 2, 3, 4, 5, 6, 0];
const zones: string[] = (() => {
  try { return (Intl as unknown as { supportedValuesOf?: (k: string) => string[] }).supportedValuesOf?.('timeZone') ?? []; } catch { return []; }
})();

export const blankSchedule: Schedule = {
  id: 0, name: '', timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC',
  weekly: [1, 2, 3, 4, 5].map((day) => ({ day, start: '09:00', end: '17:00' })), holidays: [],
};

const mins = (t: string) => { const [h, m] = t.split(':').map(Number); return (h || 0) * 60 + (m || 0); };

interface Props { s: Schedule; set: (p: Partial<Schedule>) => void }

export function ScheduleForm({ s, set }: Props) {
  const weekly = s.weekly ?? [];
  const holidays = s.holidays ?? [];
  const setRanges = (day: number, ranges: WeeklyRange[]) => set({ weekly: [...weekly.filter((r) => r.day !== day), ...ranges].sort((a, b) => a.day - b.day || mins(a.start) - mins(b.start)) });
  const copyToWeekdays = (day: number) => {
    const src = weekly.filter((r) => r.day === day);
    const others = weekly.filter((r) => r.day === 0 || r.day === 6 || r.day === day);
    const copies = [1, 2, 3, 4, 5].filter((d) => d !== day).flatMap((d) => src.map((r) => ({ ...r, day: d })));
    set({ weekly: [...others, ...copies].sort((a, b) => a.day - b.day || mins(a.start) - mins(b.start)) });
  };

  return (
    <>
      <div className="form-section">
        <div className="form-grid">
          <Field label="Name"><input className="input" value={s.name} onChange={(e) => set({ name: e.target.value })} /></Field>
          <Field label="Timezone" hint="IANA name, e.g. Europe/Rome">
            <input className="input" list="tz-list" value={s.timezone} onChange={(e) => set({ timezone: e.target.value })} />
            <datalist id="tz-list">{zones.map((z) => <option key={z} value={z} />)}</datalist>
          </Field>
        </div>
      </div>
      <div className="form-section">
        <h3>Weekly hours</h3>
        <div>
          {ORDER.map((day) => {
            const ranges = weekly.filter((r) => r.day === day);
            return (
              <div key={day} className="day-row">
                <div>
                  <div className="strong">{DAYS[day]}</div>
                  <div className="small muted">{ranges.length ? `${ranges.length} range${ranges.length > 1 ? 's' : ''}` : 'Closed'}</div>
                </div>
                <div className="col" style={{ gap: 6 }}>
                  {ranges.map((r, i) => (
                    <div key={i} className="row wrap" style={{ gap: 6 }}>
                      <input className="input sm" type="time" style={{ width: 120 }} value={r.start} aria-label={`${DAYS[day]} start`}
                        onChange={(e) => setRanges(day, ranges.map((x, j) => (j === i ? { ...x, start: e.target.value } : x)))} />
                      <span className="muted">–</span>
                      <input className="input sm" type="time" style={{ width: 120 }} value={r.end} aria-label={`${DAYS[day]} end`}
                        onChange={(e) => setRanges(day, ranges.map((x, j) => (j === i ? { ...x, end: e.target.value } : x)))} />
                      {mins(r.end) <= mins(r.start) && <span className="small" style={{ color: 'var(--danger)' }}>End must be after start</span>}
                      <button className="btn ghost icon sm" aria-label="Remove range" onClick={() => setRanges(day, ranges.filter((_, j) => j !== i))}><Trash2 /></button>
                    </div>
                  ))}
                  <div className="row" style={{ gap: 6 }}>
                    <button className="btn sm" onClick={() => setRanges(day, [...ranges, ranges.length ? { day, start: '13:00', end: '17:00' } : { day, start: '09:00', end: '17:00' }])}><Plus />Add hours</button>
                    {day >= 1 && day <= 5 && ranges.length > 0 && <button className="btn ghost sm" onClick={() => copyToWeekdays(day)}><Copy />Copy to weekdays</button>}
                  </div>
                  <div className="hours-bar" aria-hidden>
                    {ranges.map((r, i) => <span key={i} style={{ left: `${(mins(r.start) / 1440) * 100}%`, width: `${Math.max(0, mins(r.end) - mins(r.start)) / 14.4}%` }} />)}
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
      <div className="form-section">
        <h3>Holidays (closed all day)</h3>
        {holidays.length === 0 && <div className="small muted">No holidays.</div>}
        {holidays.map((h, i) => (
          <div key={i} className="row wrap">
            <input className="input sm" type="date" style={{ width: 160 }} value={h.date} aria-label="Holiday date"
              onChange={(e) => set({ holidays: holidays.map((x, j) => (j === i ? { ...x, date: e.target.value } : x)) })} />
            <input className="input sm grow" style={{ minWidth: 160 }} placeholder="Name" value={h.name} aria-label="Holiday name"
              onChange={(e) => set({ holidays: holidays.map((x, j) => (j === i ? { ...x, name: e.target.value } : x)) })} />
            <button className="btn ghost icon sm" aria-label="Remove holiday" onClick={() => set({ holidays: holidays.filter((_, j) => j !== i) })}><Trash2 /></button>
          </div>
        ))}
        <div><button className="btn sm" onClick={() => set({ holidays: [...holidays, { date: '', name: '' }] })}><Plus />Add holiday</button></div>
      </div>
    </>
  );
}
