import { Copy } from 'lucide-react';
import { toast } from './Toast';

/** Read-only value with a copy-to-clipboard button. */
export function CopyField({ value, label }: { value: string; label: string }) {
  const copy = async () => {
    try { await navigator.clipboard.writeText(value); toast.success(`${label} copied`); } catch { toast.error('Copy failed — select the text and copy it manually'); }
  };
  return (
    <div className="row">
      <input className="input mono grow" readOnly value={value} onFocus={(e) => e.target.select()} aria-label={label} />
      <button type="button" className="btn" onClick={copy}><Copy />Copy</button>
    </div>
  );
}
