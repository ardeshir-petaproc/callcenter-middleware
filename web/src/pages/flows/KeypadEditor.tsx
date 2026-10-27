import type { Action } from '../../api/types';
import { ActionEditor } from '../../components/ActionEditor';

export const KEYS = ['1', '2', '3', '4', '5', '6', '7', '8', '9', '*', '0', '#'];

interface Props { options: Record<string, Action>; onChange: (o: Record<string, Action>) => void; disabled: boolean; flowId: number }

/** 0-9 * # keypad grid; each key maps to a routing action (empty type = key unused). */
export function KeypadEditor({ options, onChange, disabled, flowId }: Props) {
  const setKey = (k: string, a: Action) => {
    const next = { ...options };
    if (!a.type) delete next[k];
    else next[k] = a;
    onChange(next);
  };
  return (
    <div className="keypad-grid">
      {KEYS.map((k) => {
        const a = options[k];
        return (
          <div key={k} className={`key-cell ${a?.type ? 'set' : ''}`}>
            <span className="key" aria-hidden>{k}</span>
            <div className="grow" style={{ minWidth: 0 }}>
              <ActionEditor value={a} compact disabled={disabled} excludeFlowId={flowId} label={`Key ${k}`} onChange={(v) => setKey(k, v)} />
            </div>
          </div>
        );
      })}
    </div>
  );
}
