import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import { Check, Mail, MailOpen, PhoneOutgoing, Trash2, Voicemail as VmIcon } from 'lucide-react';
import type { Voicemail } from '../../api/types';
import { authUrl, del, get, patch, post } from '../../api/client';
import { useApi } from '../../lib/hooks';
import { onLive } from '../../state/ws';
import { live } from '../../state/live';
import { useQueueNames } from '../../state/lookups';
import { useCan } from '../../state/auth';
import { PERMS } from '../../lib/perms';
import { fmtDateTime, fmtDuration, fmtRelative } from '../../lib/format';
import { Card, Empty, ErrorBox, PageHeader, Segmented, SkeletonRows } from '../../components/ui/primitives';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';

export default function VoicemailPage() {
  const can = useCan();
  const { data, loading, error, reload, setData } = useApi((s) => get<Voicemail[]>('/voicemails', undefined, s), []);
  const [filter, setFilter] = useState<'unheard' | 'all'>('unheard');
  const queueName = useQueueNames();
  useEffect(() => onLive('voicemail.created', () => reload()), [reload]);
  useEffect(() => { if (data) live.set({ unheardVoicemails: data.filter((v) => !v.heard).length }); }, [data]);

  const items = (data ?? []).filter((v) => filter === 'all' || !v.heard).sort((a, b) => Date.parse(b.created_at) - Date.parse(a.created_at));
  const mark = async (v: Voicemail, heard: boolean) => {
    try {
      await patch(`/voicemails/${v.id}`, { heard });
      setData((l) => l?.map((x) => (x.id === v.id ? { ...x, heard } : x)));
    } catch (e) { toast.error(e); }
  };
  const remove = async (v: Voicemail) => {
    if (!(await confirm({ title: 'Delete voicemail?', message: `Delete the message from ${v.caller_name || v.caller_number}?`, danger: true, confirmLabel: 'Delete' }))) return;
    try { await del(`/voicemails/${v.id}`); setData((l) => l?.filter((x) => x.id !== v.id)); toast.success('Voicemail deleted'); } catch (e) { toast.error(e); }
  };
  const callBack = async (v: Voicemail) => {
    try {
      await post('/callbacks', { number: v.caller_number, name: v.caller_name || undefined, queue_id: v.queue_id || undefined, notes: `From voicemail #${v.id}` });
      toast.success('Callback created');
    } catch (e) { toast.error(e); }
  };
  const unheard = (data ?? []).filter((v) => !v.heard).length;

  return (
    <>
      <PageHeader title="Voicemail" sub={`${unheard} unheard message${unheard === 1 ? '' : 's'}`} />
      <Card flush title={<Segmented value={filter} onChange={setFilter} label="Filter" options={[{ value: 'unheard', label: `Unheard (${unheard})` }, { value: 'all', label: 'All' }]} />}>
        {error ? <div style={{ padding: 16 }}><ErrorBox error={error} onRetry={reload} /></div>
          : loading && !data ? <SkeletonRows rows={5} />
          : items.length === 0 ? <Empty title={filter === 'unheard' ? 'All caught up' : 'No voicemail'} icon={<VmIcon />} />
          : (
            <div className="list">
              {items.map((v) => (
                <div key={v.id} className="list-item" style={{ alignItems: 'flex-start', flexWrap: 'wrap' }}>
                  <span className="avatar" style={v.heard ? { background: 'var(--surface-3)', color: 'var(--text-3)' } : undefined}>{v.heard ? <MailOpen size={16} /> : <Mail size={16} />}</span>
                  <div className="grow col" style={{ gap: 6, minWidth: 220 }}>
                    <div className="row wrap">
                      <span className="strong">{v.caller_name || v.caller_number}</span>
                      {v.caller_name && <span className="small muted">{v.caller_number}</span>}
                      {!v.heard && <span className="badge info">New</span>}
                      <span className="small muted" title={fmtDateTime(v.created_at)}>{fmtRelative(v.created_at)} · {fmtDuration(v.duration_sec)}{v.queue_id ? ` · ${queueName(v.queue_id)}` : ''}</span>
                    </div>
                    <audio controls preload="none" src={authUrl(`/voicemails/${v.id}/audio`)} onPlay={() => { if (!v.heard) void mark(v, true); }} />
                    {v.transcript ? <div className="small pre" style={{ background: 'var(--surface-2)', padding: 8, borderRadius: 6 }}>“{v.transcript}”</div> : <div className="small muted">No transcript</div>}
                    {v.call_id && <Link className="small" to={`/calls/${encodeURIComponent(v.call_id)}`}>View call</Link>}
                  </div>
                  <div className="btn-group">
                    <button className="btn sm" onClick={() => callBack(v)}><PhoneOutgoing />Callback</button>
                    <button className="btn sm" onClick={() => mark(v, !v.heard)}><Check />{v.heard ? 'Mark unheard' : 'Mark heard'}</button>
                    {can(PERMS.callsControl) && <button className="btn sm icon" aria-label="Delete" onClick={() => remove(v)}><Trash2 /></button>}
                  </div>
                </div>
              ))}
            </div>
          )}
      </Card>
    </>
  );
}
