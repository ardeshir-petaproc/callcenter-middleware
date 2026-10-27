import type { ReactNode } from 'react';
import type { Action, ActionType } from '../api/types';
import { useLookup } from '../state/lookups';
import { humanize } from '../lib/format';

export const ACTION_TYPES: { value: ActionType; label: string }[] = [
  { value: '', label: 'Do nothing' },
  { value: 'queue', label: 'Send to queue' },
  { value: 'ai', label: 'AI agent' },
  { value: 'voicemail', label: 'Voicemail' },
  { value: 'callback', label: 'Offer callback' },
  { value: 'agent', label: 'Ring agent' },
  { value: 'external', label: 'External number' },
  { value: 'flow', label: 'Go to IVR flow' },
  { value: 'dialplan', label: 'Dialplan extension' },
  { value: 'hangup', label: 'Hang up' },
];

const NEEDS_TARGET = new Set<ActionType>(['queue', 'ai', 'agent', 'external', 'flow', 'dialplan']);
const OPTIONAL_TARGET = new Set<ActionType>(['voicemail', 'callback']);

interface Props {
  value: Action | undefined;
  onChange: (a: Action) => void;
  disabled?: boolean;
  allowMedia?: boolean;
  compact?: boolean;
  excludeFlowId?: number;
  label?: string;
}

/** Reusable routing-action editor: type select → matching target picker (+ optional media prompt). */
export function ActionEditor({ value, onChange, disabled, allowMedia = true, compact, excludeFlowId, label }: Props) {
  const a: Action = value ?? { type: '' };
  const queues = useLookup('queues');
  const profiles = useLookup('aiProfiles');
  const flows = useLookup('flows');
  const users = useLookup('users');
  const set = (p: Partial<Action>) => onChange({ ...a, ...p });
  const setType = (type: ActionType) => onChange({ type, target: '', media: a.media });
  const t = a.target ?? '';
  const sel = (opts: { id: number; name: string }[], placeholder: string) => (
    <select className={`select ${compact ? 'sm' : ''}`} value={t} disabled={disabled} onChange={(e) => set({ target: e.target.value })} aria-label={placeholder}>
      <option value="">{placeholder}</option>
      {opts.map((o) => <option key={o.id} value={String(o.id)}>{o.name}</option>)}
      {t && !opts.some((o) => String(o.id) === t) && <option value={t}>#{t} (missing)</option>}
    </select>
  );

  let target: ReactNode = null;
  switch (a.type) {
    case 'queue': target = sel(queues, 'Select queue…'); break;
    case 'voicemail': case 'callback': target = sel(queues, 'Default queue'); break;
    case 'ai': target = sel(profiles, 'Select AI profile…'); break;
    case 'flow': target = sel(flows.filter((f) => f.id !== excludeFlowId), 'Select flow…'); break;
    case 'agent': target = sel(users.map((u) => ({ id: u.id, name: `${u.display_name || u.username}${u.extension ? ` (${u.extension})` : ''}` })), 'Select agent…'); break;
    case 'external':
      target = <input className={`input ${compact ? 'sm' : ''}`} inputMode="tel" placeholder="+15551234567" value={t} disabled={disabled} onChange={(e) => set({ target: e.target.value })} aria-label="Phone number" />;
      break;
    case 'dialplan':
      target = <input className={`input ${compact ? 'sm' : ''}`} placeholder="extension@context" value={t} disabled={disabled} onChange={(e) => set({ target: e.target.value })} aria-label="Dialplan extension" />;
      break;
    default: break;
  }
  const invalid = NEEDS_TARGET.has(a.type) && !t;

  return (
    <div className="col" style={{ gap: 6, minWidth: 0 }} aria-label={label}>
      <div className="row wrap" style={{ gap: 6 }}>
        <select className={`select ${compact ? 'sm' : ''}`} style={{ flex: '1 1 140px' }} value={a.type} disabled={disabled} onChange={(e) => setType(e.target.value as ActionType)} aria-label={label ? `${label} action` : 'Action'}>
          {ACTION_TYPES.map((o) => <option key={o.value} value={o.value}>{o.label}</option>)}
        </select>
        {target && <div style={{ flex: '1 1 160px', minWidth: 0 }}>{target}</div>}
      </div>
      {allowMedia && a.type !== '' && (
        <input className={`input ${compact ? 'sm' : ''}`} placeholder={compact ? "Prompt media (optional)" : "Prompt before action (optional), e.g. sound:custom/transfer"} value={a.media ?? ''} disabled={disabled}
          onChange={(e) => set({ media: e.target.value })} aria-label="Prompt media" />
      )}
      {invalid && <span className="small" style={{ color: 'var(--danger)' }}>Choose a target</span>}
      {OPTIONAL_TARGET.has(a.type) && !compact && <span className="hint small muted">Queue is optional for {humanize(a.type).toLowerCase()}.</span>}
    </div>
  );
}

/** One-line human description of an action, used in previews. */
export function useDescribeAction(): (a: Action | undefined) => string {
  const queues = useLookup('queues');
  const profiles = useLookup('aiProfiles');
  const flows = useLookup('flows');
  const users = useLookup('users');
  return (a) => {
    if (!a || !a.type) return 'Nothing';
    const t = a.target ?? '';
    const name = (list: { id: number; name?: string; display_name?: string; username?: string }[]) => {
      const x = list.find((i) => String(i.id) === t);
      return x ? x.name ?? x.display_name ?? x.username ?? t : t ? `#${t}` : '';
    };
    switch (a.type) {
      case 'queue': return `Queue: ${name(queues)}`;
      case 'ai': return `AI: ${name(profiles)}`;
      case 'flow': return `Flow: ${name(flows)}`;
      case 'agent': return `Agent: ${name(users)}`;
      case 'external': return `Dial ${t}`;
      case 'dialplan': return `Dialplan ${t}`;
      case 'voicemail': return t ? `Voicemail (${name(queues)})` : 'Voicemail';
      case 'callback': return t ? `Callback (${name(queues)})` : 'Offer callback';
      case 'hangup': return 'Hang up';
      default: return humanize(a.type);
    }
  };
}
