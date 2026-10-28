import { useRef, useState } from 'react';
import { Circle, Ear, Headphones, Megaphone, Pause, PhoneForwarded, PhoneOff, Play, MessageSquareText } from 'lucide-react';
import type { CallView } from '../../api/types';
import { post } from '../../api/client';
import { toast } from '../ui/Toast';
import { confirm } from '../ui/Confirm';
import { TransferDialog } from './TransferDialog';
import { Popover } from '../ui/Popover';
import { useCan } from '../../state/auth';
import { PERMS } from '../../lib/perms';

interface Props {
  call: CallView;
  compact?: boolean;
  onTranscript?: () => void;
  monitor?: boolean;
}

const ANSWERED = new Set(['connected', 'hold']);

export function CallActions({ call, compact, onTranscript, monitor = true }: Props) {
  const can = useCan();
  const [transfer, setTransfer] = useState(false);
  const [busy, setBusy] = useState<string | null>(null);
  const id = encodeURIComponent(call.id);
  const cls = compact ? 'btn sm icon' : 'btn';

  const run = async (key: string, fn: () => Promise<unknown>, ok?: string) => {
    setBusy(key);
    try { await fn(); if (ok) toast.success(ok); } catch (e) { toast.error(e); } finally { setBusy(null); }
  };

  const hangup = async () => {
    const ok = await confirm({ title: 'Hang up call?', message: `End the call with ${call.caller_name || call.caller_number}?`, confirmLabel: 'Hang up', danger: true });
    if (ok) void run('hangup', () => post(`/calls/${id}/hangup`), 'Call ended');
  };
  const onHold = call.state === 'hold';
  const answered = ANSWERED.has(call.state);
  const label = (t: string) => (compact ? undefined : t);

  return (
    <div className="btn-group" onClick={(e) => e.stopPropagation()}>
      {onTranscript && (
        <button className={cls} title="Live transcript" aria-label="Live transcript" onClick={onTranscript}><MessageSquareText />{label('Transcript')}</button>
      )}
      <button className={cls} disabled={!answered || busy === 'hold'} title={onHold ? 'Resume' : 'Hold'} aria-label={onHold ? 'Resume' : 'Hold'}
        onClick={() => run('hold', () => post(`/calls/${id}/hold`, { on: !onHold }), onHold ? 'Call resumed' : 'Call on hold')}>
        {onHold ? <Play /> : <Pause />}{label(onHold ? 'Resume' : 'Hold')}
      </button>
      <button className={cls} disabled={!answered || busy === 'record'} title="Start recording" aria-label="Start recording"
        onClick={() => run('record', () => post(`/calls/${id}/record`), 'Recording started')}>
        <Circle style={{ color: 'var(--danger-solid)' }} />{label('Record')}
      </button>
      <button className={cls} title="Transfer" aria-label="Transfer" onClick={() => setTransfer(true)}><PhoneForwarded />{label('Transfer')}</button>
      {monitor && can(PERMS.callsMonitor) && <MonitorMenu callId={call.id} compact={compact} disabled={!answered} onRun={run} />}
      <button className={`${cls} danger`} disabled={busy === 'hangup'} title="Hang up" aria-label="Hang up" onClick={hangup}><PhoneOff />{label('Hang up')}</button>
      {transfer && <TransferDialog callId={call.id} open={transfer} onClose={() => setTransfer(false)} />}
    </div>
  );
}

const MODES = [
  { mode: 'spy', label: 'Listen (spy)', hint: 'Silent listen', icon: Ear },
  { mode: 'whisper', label: 'Whisper', hint: 'Only the agent hears you', icon: Headphones },
  { mode: 'barge', label: 'Barge in', hint: 'Join the conversation', icon: Megaphone },
] as const;

function MonitorMenu({ callId, compact, disabled, onRun }: {
  callId: string; compact?: boolean; disabled: boolean; onRun: (k: string, fn: () => Promise<unknown>, ok?: string) => Promise<void>;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLButtonElement>(null);
  return (
    <>
      <button ref={ref} className={compact ? 'btn sm icon' : 'btn'} disabled={disabled} aria-haspopup="menu" aria-expanded={open}
        title="Monitor" aria-label="Monitor" onClick={() => setOpen((o) => !o)}>
        <Headphones />{compact ? null : 'Monitor'}
      </button>
      <Popover anchor={ref} open={open} onClose={() => setOpen(false)}>
        <div className="menu-head small muted">Rings your extension</div>
        {MODES.map((m) => (
          <button key={m.mode} role="menuitem" onClick={() => {
            setOpen(false);
            void onRun('monitor', () => post(`/calls/${encodeURIComponent(callId)}/monitor`, { mode: m.mode }), `${m.label}: ringing your phone`);
          }}>
            <m.icon />
            <span className="col" style={{ gap: 0 }}><span>{m.label}</span><span className="small muted">{m.hint}</span></span>
          </button>
        ))}
      </Popover>
    </>
  );
}
