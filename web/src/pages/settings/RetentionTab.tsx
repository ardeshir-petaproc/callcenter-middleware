import { useEffect, useState } from 'react';
import { Archive, Download, Play, Save } from 'lucide-react';
import { downloadFile, get, post, put } from '../../api/client';
import { PERMS } from '../../lib/perms';
import { useCan } from '../../state/auth';
import type { RetentionPolicy, RetentionResult } from '../../api/types_routing';
import { useApi } from '../../lib/hooks';
import { Card, ErrorBox, Field, SkeletonRows, Toggle } from '../../components/ui/primitives';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';

const FIELDS: { key: keyof RetentionPolicy; label: string; hint: string }[] = [
  { key: 'recording_days', label: 'Call recordings', hint: 'Audio files are deleted after this many days' },
  { key: 'transcript_days', label: 'Transcripts', hint: 'Call transcripts are deleted after this many days' },
  { key: 'email_days', label: 'Imported emails', hint: 'Messages imported by email connectors' },
];

/** Data retention policy (runs hourly) plus the agent playback setting. */
export function RetentionTab() {
  const pol = useApi((s) => get<RetentionPolicy>('/retention', undefined, s), []);
  const settings = useApi((s) => get<Record<string, string>>('/settings', undefined, s), []);
  const [draft, setDraft] = useState<RetentionPolicy>({ recording_days: 0, transcript_days: 0, email_days: 0 });
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<RetentionResult | null>(null);
  const canExport = useCan()(PERMS.auditView);
  useEffect(() => { if (pol.data) setDraft(pol.data); }, [pol.data]);
  const dirty = JSON.stringify(draft) !== JSON.stringify(pol.data);
  const playback = (settings.data?.['recordings.agent_playback'] ?? '1') !== '0';

  const save = async () => {
    setBusy(true);
    try { pol.setData(await put<RetentionPolicy>('/retention', draft)); toast.success('Retention policy saved'); } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  const run = async () => {
    if (!(await confirm({ title: 'Apply retention now?', message: 'Recordings, transcripts and emails older than the policy are permanently deleted.', danger: true, confirmLabel: 'Run now' }))) return;
    setBusy(true);
    try { setResult(await post<RetentionResult>('/retention/run')); toast.success('Retention applied'); } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  const exportOrg = async () => {
    setBusy(true);
    try { await downloadFile('/export/organization.zip', 'organization.zip'); toast.success('Export downloaded'); } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  const setPlayback = async (on: boolean) => {
    try { settings.setData(await put<Record<string, string>>('/settings', { 'recordings.agent_playback': on ? '1' : '0' })); toast.success('Saved'); } catch (e) { toast.error(e); }
  };

  if (pol.error) return <ErrorBox error={pol.error} onRetry={pol.reload} />;
  if (pol.loading && !pol.data) return <div className="card"><SkeletonRows rows={4} /></div>;
  return (
    <div className="stack">
      <Card title="Data retention" icon={<Archive />} flush>
        <div className="card-body col gap-lg">
          <p className="small muted">Older data is purged automatically every hour and every purge is audited. 0 keeps data forever.</p>
          <div className="form-grid">
            {FIELDS.map((f) => (
              <Field key={f.key} label={`${f.label} (days)`} hint={f.hint}>
                <input className="input" type="number" min={0} value={draft[f.key]} onChange={(e) => setDraft({ ...draft, [f.key]: Math.max(0, Number(e.target.value)) })} />
              </Field>
            ))}
          </div>
          {result && (
            <div className="alert info"><div>Last run deleted <strong>{result.recordings}</strong> recordings, <strong>{result.transcripts}</strong> transcripts and <strong>{result.emails}</strong> emails{result.failed ? `; ${result.failed} failed` : ''}.</div></div>
          )}
        </div>
        <div className="card-foot">
          <button className="btn" disabled={busy || dirty} title={dirty ? 'Save the policy first' : undefined} onClick={run}><Play />Run now</button>
          <button className="btn primary" disabled={busy || !dirty} onClick={save}><Save />Save policy</button>
        </div>
      </Card>
      <Card title="Recording playback">
        <Toggle checked={playback} disabled={!settings.data} onChange={setPlayback} label="Agents may play the recordings of their own calls" />
      </Card>
      {canExport && (
        <Card title="Organization data export" actions={<button className="btn" disabled={busy} onClick={exportOrg}><Download />Export organization data</button>}>
          <p className="small muted">Downloads a ZIP archive with this organization's configuration and records (for backups, audits or data requests).</p>
        </Card>
      )}
    </div>
  );
}
