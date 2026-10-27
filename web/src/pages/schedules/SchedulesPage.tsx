import { useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { CalendarClock } from 'lucide-react';
import type { Schedule } from '../../api/types';
import { useCrud } from '../../lib/crud';
import { useDraft } from '../../lib/hooks';
import { ConfigLayout, EditorFooter } from '../../components/ConfigLayout';
import { Card, Empty } from '../../components/ui/primitives';
import { confirm } from '../../components/ui/Confirm';
import { toast } from '../../components/ui/Toast';
import { blankSchedule, ScheduleForm } from './ScheduleForm';

export default function SchedulesPage() {
  const { id } = useParams();
  const nav = useNavigate();
  const crud = useCrud<Schedule>('/schedules', 'schedules');
  const isNew = id === 'new';
  const source = useMemo(() => {
    if (isNew) return { ...blankSchedule };
    const s = crud.items.find((x) => String(x.id) === id);
    return s ? { ...s, weekly: s.weekly ?? [], holidays: s.holidays ?? [] } : undefined;
  }, [isNew, id, crud.items]);
  const { draft, set, dirty } = useDraft<Schedule>(source, id ?? '');
  const [busy, setBusy] = useState(false);

  const save = async () => {
    if (!draft) return;
    if (!draft.name.trim()) { toast.error('Name is required'); return; }
    if (draft.weekly.some((r) => r.end <= r.start)) { toast.error('Each range must end after it starts'); return; }
    if (draft.holidays.some((h) => !h.date)) { toast.error('Every holiday needs a date'); return; }
    setBusy(true);
    try {
      const saved = await crud.save(isNew ? { ...draft, id: undefined } as Partial<Schedule> : draft);
      toast.success(isNew ? 'Schedule created' : 'Schedule saved');
      if (isNew) nav(`/schedules/${saved.id}`, { replace: true });
    } catch (e) { toast.error(e); } finally { setBusy(false); }
  };
  const remove = async () => {
    if (!draft || !(await confirm({ title: 'Delete schedule?', message: `Delete "${draft.name}"? Flows using it will be treated as always open.`, danger: true, confirmLabel: 'Delete' }))) return;
    try { await crud.remove(draft.id); toast.success('Schedule deleted'); nav('/schedules', { replace: true }); } catch (e) { toast.error(e); }
  };

  return (
    <ConfigLayout
      title="Schedules" sub="Business hours and holidays used by IVR flows" basePath="/schedules"
      items={crud.items} loading={crud.loading} error={crud.error} onRetry={crud.reload}
      selected={id} canCreate newLabel="New schedule" emptyIcon={<CalendarClock />}
      search={(s) => s.name}
      renderItem={(s) => (
        <div className="grow">
          <div className="strong ellipsis">{s.name}</div>
          <div className="small muted">{s.timezone} · {(s.holidays ?? []).length} holidays</div>
        </div>
      )}
    >
      {!id ? <div className="card"><Empty title="Select a schedule" icon={<CalendarClock />}>Pick a schedule or create one.</Empty></div>
        : !draft ? <div className="card"><Empty title={crud.loading ? 'Loading…' : 'Schedule not found'} /></div>
        : (
          <Card title={isNew ? 'New schedule' : draft.name || 'Schedule'} flush>
            <div className="card-body"><ScheduleForm s={draft} set={set} /></div>
            <EditorFooter onSave={save} onDelete={remove} busy={busy} canEdit dirty={dirty} isNew={isNew} />
          </Card>
        )}
    </ConfigLayout>
  );
}
