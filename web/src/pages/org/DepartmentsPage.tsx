import { useParams } from 'react-router-dom';
import { Building2 } from 'lucide-react';
import type { Department } from '../../api/types_org';
import { useCrud } from '../../lib/crud';
import { useEntityEditor } from '../../lib/editor';
import { PERMS } from '../../lib/perms';
import { useCan } from '../../state/auth';
import { useUserNames } from '../../state/lookups';
import { ConfigLayout, EditorFooter } from '../../components/ConfigLayout';
import { RevisionsButton } from '../../components/RevisionsButton';
import { Card, Empty } from '../../components/ui/primitives';
import { DepartmentForm } from './DepartmentForm';

const blank: Department = {
  id: 0, name: '', description: '', manager_id: 0, schedule_id: 0, capacity: 0, priority: 0,
  fallback: { type: 'voicemail' }, languages: [], enabled: true,
};

export default function DepartmentsPage() {
  const { id } = useParams();
  const canEdit = useCan()(PERMS.orgManage);
  const crud = useCrud<Department>('/departments', 'departments');
  const userName = useUserNames();
  const ed = useEntityEditor({
    crud, id, blank, base: '/org/departments', noun: 'Department', label: (d) => d.name,
    validate: (d) => (d.name.trim() ? null : 'Name is required'),
    normalize: (d) => ({ ...d, languages: d.languages ?? [], fallback: d.fallback ?? { type: '' } }),
  });
  const d = ed.draft;

  return (
    <ConfigLayout
      title="Departments" sub="Business units that own requests: hours, capacity, languages and fallback" basePath="/org/departments"
      items={crud.items} loading={crud.loading} error={crud.error} onRetry={crud.reload}
      selected={id} canCreate={canEdit} newLabel="New department" emptyIcon={<Building2 />}
      search={(x) => `${x.name} ${x.description}`}
      renderItem={(x) => (
        <div className="grow">
          <div className="strong ellipsis" dir="auto">{x.name}</div>
          <div className="small muted ellipsis">
            <span dir="auto">{x.description || '—'}</span>{x.manager_id ? ` · ${userName(x.manager_id)}` : ''}{x.enabled ? '' : ' · disabled'}
          </div>
        </div>
      )}
    >
      {!id ? <div className="card"><Empty title="Select a department" icon={<Building2 />}>Departments group queues and employees, e.g. Sales or Support.</Empty></div>
        : !d ? <div className="card"><Empty title={crud.loading ? 'Loading…' : 'Department not found'} /></div>
        : (
          <Card title={ed.isNew ? 'New department' : d.name || 'Department'} flush>
            <div className="card-body"><DepartmentForm d={d} set={ed.set} disabled={!canEdit} /></div>
            <EditorFooter onSave={ed.save} onDelete={ed.remove} busy={ed.busy} canEdit={canEdit} dirty={ed.dirty} isNew={ed.isNew}
              extra={!ed.isNew && <RevisionsButton entity="department" id={d.id} />} />
          </Card>
        )}
    </ConfigLayout>
  );
}
