// Human-friendly formatting helpers.

export function secsSince(iso: string | undefined | null, now: number): number {
  if (!iso) return 0;
  const t = Date.parse(iso);
  if (Number.isNaN(t) || t <= 0) return 0;
  return Math.max(0, Math.floor((now - t) / 1000));
}

/** 75 -> "1:15", 3725 -> "1:02:05" */
export function fmtDuration(totalSecs: number | undefined | null): string {
  const s = Math.max(0, Math.round(totalSecs ?? 0));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  const pad = (n: number) => String(n).padStart(2, '0');
  return h > 0 ? `${h}:${pad(m)}:${pad(sec)}` : `${m}:${pad(sec)}`;
}

/** 3725 -> "1h 2m", 75 -> "1m 15s" */
export function fmtLongDuration(totalSecs: number | undefined | null): string {
  const s = Math.max(0, Math.round(totalSecs ?? 0));
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  if (m > 0) return `${m}m ${s % 60}s`;
  return `${s}s`;
}

const rtf = typeof Intl !== 'undefined' && 'RelativeTimeFormat' in Intl
  ? new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' }) : null;

export function fmtRelative(iso: string | undefined | null, now = Date.now()): string {
  if (!iso) return '—';
  const t = Date.parse(iso);
  if (Number.isNaN(t) || t <= 0) return '—';
  const diff = Math.round((t - now) / 1000);
  const abs = Math.abs(diff);
  if (!rtf) return new Date(t).toLocaleString();
  if (abs < 45) return diff <= 0 ? 'just now' : 'in a few seconds';
  if (abs < 3600) return rtf.format(Math.round(diff / 60), 'minute');
  if (abs < 86400) return rtf.format(Math.round(diff / 3600), 'hour');
  if (abs < 86400 * 30) return rtf.format(Math.round(diff / 86400), 'day');
  return new Date(t).toLocaleDateString();
}

export function fmtDateTime(iso: string | undefined | null): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime()) || d.getFullYear() < 1971) return '—';
  return d.toLocaleString(undefined, { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
}

export function fmtTime(iso: string | undefined | null): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime()) || d.getFullYear() < 1971) return '—';
  return d.toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

const nf = new Intl.NumberFormat();
export function fmtNum(n: number | undefined | null): string {
  if (n === undefined || n === null || Number.isNaN(n)) return '—';
  return nf.format(n);
}

/** Accepts 0..1 or 0..100 and renders "92.5%". */
export function fmtPct(n: number | undefined | null): string {
  if (n === undefined || n === null || Number.isNaN(n)) return '—';
  const v = n <= 1 && n > 0 ? n * 100 : n;
  return `${v >= 99.95 || v === 0 ? v.toFixed(0) : v.toFixed(1)}%`;
}
export function pctValue(n: number | undefined | null): number {
  if (!n) return 0;
  return n <= 1 ? n * 100 : n;
}

export function humanize(s: string | undefined | null): string {
  if (!s) return '—';
  const t = s.replace(/[_.-]+/g, ' ').trim();
  return t.charAt(0).toUpperCase() + t.slice(1);
}

export function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return '?';
  return (parts[0][0] + (parts.length > 1 ? parts[parts.length - 1][0] : '')).toUpperCase();
}

export function toLocalDateInput(d: Date): string {
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}

/** YYYY-MM-DD local day bounds -> RFC3339 instants. */
export function dayStartIso(day: string): string { return new Date(`${day}T00:00:00`).toISOString(); }
export function dayEndIso(day: string): string { return new Date(`${day}T23:59:59.999`).toISOString(); }
