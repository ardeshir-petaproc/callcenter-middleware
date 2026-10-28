import { get } from '../../api/client';
import type { JobTemplate } from '../../api/types_org';
import { useApi } from '../../lib/hooks';
import { Dialog } from '../../components/ui/Dialog';
import { ErrorBox, SkeletonRows } from '../../components/ui/primitives';

/** Ready-made job descriptions (English and Persian titles) to start a new job profile from. */
export function TemplatePicker({ open, onClose, onPick }: { open: boolean; onClose: () => void; onPick: (t: JobTemplate) => void }) {
  const { data, loading, error, reload } = useApi((s) => get<JobTemplate[]>('/job-profile-templates', undefined, s), [], open);
  return (
    <Dialog open={open} onClose={onClose} title="Start from a template" size="wide">
      {error ? <ErrorBox error={error} onRetry={reload} />
        : loading && !data ? <SkeletonRows rows={4} />
        : (
          <div className="list" style={{ border: '1px solid var(--border)', borderRadius: 8 }}>
            {(data ?? []).map((t) => (
              <button key={t.title} type="button" className="list-item clickable plain-button" onClick={() => onPick(t)}>
                <div className="grow" style={{ textAlign: 'left' }}>
                  <div className="row wrap"><span className="strong">{t.title}</span><span className="muted" dir="rtl">{t.title_fa}</span></div>
                  <div className="small muted">{t.description}</div>
                  <div className="chips" style={{ marginTop: 4 }}>{(t.request_types ?? []).map((r) => <span key={r} className="chip">{r}</span>)}</div>
                </div>
              </button>
            ))}
          </div>
        )}
    </Dialog>
  );
}
