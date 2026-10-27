import { useState, type KeyboardEvent } from 'react';
import { X } from 'lucide-react';

interface Props {
  value: string[];
  onChange: (v: string[]) => void;
  placeholder?: string;
  suggestions?: string[];
  disabled?: boolean;
  id?: string;
}

/** Chip-style multi-value input: Enter/comma adds, Backspace removes the last chip. */
export function TagInput({ value, onChange, placeholder, suggestions, disabled, id }: Props) {
  const [text, setText] = useState('');
  const tags = value ?? [];
  const add = (raw: string) => {
    const t = raw.trim();
    if (!t || tags.includes(t)) { setText(''); return; }
    onChange([...tags, t]);
    setText('');
  };
  const onKey = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' || e.key === ',') { e.preventDefault(); add(text); }
    else if (e.key === 'Backspace' && !text && tags.length) onChange(tags.slice(0, -1));
  };
  const listId = id ? `${id}-list` : undefined;
  return (
    <div className="tag-input">
      {tags.map((t) => (
        <span className="chip" key={t}>
          {t}
          {!disabled && <button type="button" aria-label={`Remove ${t}`} onClick={() => onChange(tags.filter((x) => x !== t))}><X /></button>}
        </span>
      ))}
      {!disabled && (
        <input
          id={id}
          value={text}
          list={suggestions?.length ? listId : undefined}
          placeholder={tags.length ? '' : placeholder ?? 'Type and press Enter'}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={onKey}
          onBlur={() => text && add(text)}
        />
      )}
      {suggestions?.length ? (
        <datalist id={listId}>{suggestions.filter((s) => !tags.includes(s)).map((s) => <option key={s} value={s} />)}</datalist>
      ) : null}
    </div>
  );
}

/** Multi-select rendered as toggleable chips. */
export function ChipSelect({ options, value, onChange, disabled }: {
  options: { value: string; label: string }[]; value: string[]; onChange: (v: string[]) => void; disabled?: boolean;
}) {
  const set = new Set(value ?? []);
  return (
    <div className="chips">
      {options.map((o) => {
        const on = set.has(o.value);
        return (
          <button
            key={o.value}
            type="button"
            disabled={disabled}
            className={`btn sm ${on ? 'active' : ''}`}
            aria-pressed={on}
            onClick={() => onChange(on ? value.filter((x) => x !== o.value) : [...(value ?? []), o.value])}
          >
            {o.label}
          </button>
        );
      })}
    </div>
  );
}
