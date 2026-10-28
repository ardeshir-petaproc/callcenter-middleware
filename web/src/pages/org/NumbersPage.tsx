import { useParams } from 'react-router-dom';
import { Phone } from 'lucide-react';
import type { PhoneNumber } from '../../api/types_org';
import { useCrud } from '../../lib/crud';
import { useEntityEditor } from '../../lib/editor';
import { PERMS } from '../../lib/perms';
import { useCan } from '../../state/auth';
import { useLookup } from '../../state/lookups';
import { ConfigLayout, EditorFooter } from '../../components/ConfigLayout';
import { RevisionsButton } from '../../components/RevisionsButton';
import { Card, Empty, Field, Toggle } from '../../components/ui/primitives';

const blank: PhoneNumber = { id: 0, number: '', label: '', flow_id: 0, enabled: true };

export default function NumbersPage() {
  const { id } = useParams();
  const canEdit = useCan()(PERMS.flowsManage);
  const crud = useCrud<PhoneNumber>('/numbers');
  const flows = useLookup('flows');
  const flowName = (fid: number) => flows.find((f) => f.id === fid)?.name ?? (fid ? `Flow #${fid}` : 'No flow');
  const ed = useEntityEditor({
    crud, id, blank, base: '/org/numbers', noun: 'Number', label: (n) => n.number,
    validate: (n) => (n.number.trim() ? null : 'Number is required'),
  });
  const n = ed.draft;
  const disabled = !canEdit;

  return (
    <ConfigLayout
      title="Phone numbers" sub="Dialed numbers (DNIS) of this organization and the IVR flow that answers them" basePath="/org/numbers"
      items={crud.items} loading={crud.loading} error={crud.error} onRetry={crud.reload}
      selected={id} canCreate={canEdit} newLabel="Add number" emptyIcon={<Phone />}
      search={(x) => `${x.number} ${x.label}`}
      renderItem={(x) => (
        <div className="grow">
          <div className="strong mono ellipsis">{x.number}</div>
          <div className="small muted ellipsis"><span dir="auto">{x.label || '—'}</span> · {flowName(x.flow_id)}{x.enabled ? '' : ' · disabled'}</div>
        </div>
      )}
    >
      {!id ? <div className="card"><Empty title="Select a number" icon={<Phone />}>A number belongs to exactly one organization across the platform.</Empty></div>
        : !n ? <div className="card"><Empty title={crud.loading ? 'Loading…' : 'Number not found'} /></div>
        : (
          <Card title={ed.isNew ? 'New phone number' : n.number} flush>
            <div className="card-body">
              <div className="form-grid">
                <Field label="Number" hint="Unique across all organizations"><input className="input mono" inputMode="tel" value={n.number} disabled={disabled} onChange={(e) => ed.set({ number: e.target.value })} /></Field>
                <Field label="Label"><input className="input" dir="auto" value={n.label} disabled={disabled} onChange={(e) => ed.set({ label: e.target.value })} /></Field>
                <Field label="Entry flow">
                  <select className="select" value={n.flow_id || ''} disabled={disabled} onChange={(e) => ed.set({ flow_id: Number(e.target.value) || 0 })}>
                    <option value="">Default routing</option>
                    {flows.map((f) => <option key={f.id} value={f.id}>{f.name}</option>)}
                  </select>
                </Field>
                <div className="field" style={{ justifyContent: 'flex-end' }}><Toggle checked={n.enabled} disabled={disabled} onChange={(v) => ed.set({ enabled: v })} label="Enabled" /></div>
              </div>
            </div>
            <EditorFooter onSave={ed.save} onDelete={ed.remove} busy={ed.busy} canEdit={canEdit} dirty={ed.dirty} isNew={ed.isNew}
              extra={!ed.isNew && <RevisionsButton entity="phone_number" id={n.id} />} />
          </Card>
        )}
    </ConfigLayout>
  );
}
