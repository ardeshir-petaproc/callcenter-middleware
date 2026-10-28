import { PhoneForwarded, Users, X } from 'lucide-react';
import type { CallView } from '../../api/types';
import { post } from '../../api/client';
import { useUserNames } from '../../state/lookups';

const label = { ringing: 'Consult ringing…', talking: 'Consulting', conference: 'Conference' } as const;

/** Shown while a warm transfer is in progress: state plus complete / conference / cancel. */
export function ConsultControls({ call, compact, busy, run }: {
  call: CallView; compact?: boolean; busy: string | null; run: (k: string, fn: () => Promise<unknown>, ok?: string) => Promise<void>;
}) {
  const userName = useUserNames();
  const id = encodeURIComponent(call.id);
  const cls = compact ? 'btn sm' : 'btn';
  const state = call.consult!;
  const step = (k: string, path: string, ok: string) => run(k, () => post(`/calls/${id}/consult/${path}`), ok);
  return (
    <>
      <span className={`badge ${state === 'ringing' ? 'warn pulse' : state === 'conference' ? 'violet' : 'info'}`} title="Warm transfer in progress">
        <span className="dot" />{label[state] ?? state}{call.consult_agent_id ? ` · ${userName(call.consult_agent_id)}` : ''}
      </span>
      <button className={`${cls} success`} disabled={busy !== null || state === 'ringing'} title="Hand the caller to the consulted party" onClick={() => step('complete', 'complete', 'Transfer completed')}>
        <PhoneForwarded />{compact ? null : 'Complete transfer'}
      </button>
      {state !== 'conference' && (
        <button className={cls} disabled={busy !== null || state === 'ringing'} title="Three-way call" onClick={() => step('conference', 'conference', 'Conference started')}>
          <Users />{compact ? null : 'Conference'}
        </button>
      )}
      <button className={cls} disabled={busy !== null} title="Drop the consulted party and return to the caller" onClick={() => step('cancel', 'cancel', 'Consult cancelled')}>
        <X />{compact ? null : 'Cancel consult'}
      </button>
    </>
  );
}
