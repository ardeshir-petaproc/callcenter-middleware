import { useState } from 'react';
import { Link, useParams } from 'react-router-dom';
import { IdCard, LayoutTemplate, Sparkles } from 'lucide-react';
import { post } from '../../api/client';
import type { JobProfile, JobTemplate } from '../../api/types_org';
import type { GenerateResult } from '../../api/types_routing';
import { useCrud } from '../../lib/crud';
import { useEntityEditor } from '../../lib/editor';
import { PERMS } from '../../lib/perms';
import { fmtDateTime } from '../../lib/format';
import { useCan } from '../../state/auth';
import { useDepartmentNames } from '../../state/lookups';
import { ConfigLayout, EditorFooter } from '../../components/ConfigLayout';
import { RevisionsButton } from '../../components/RevisionsButton';
import { Card, Empty } from '../../components/ui/primitives';
import { toast } from '../../components/ui/Toast';
import { JobProfileForm } from './JobProfileForm';
import { TemplatePicker } from './TemplatePicker';

const blank: JobProfile = {
  id: 0, title: '', department_id: 0, description: '', responsibilities: [], request_types: [], authority_limits: '',
  escalation_conditions: '', escalate_to_user_id: 0, is_template: false, version: 0, updated_at: '',
};

export default function JobProfilesPage() {
  const { id } = useParams();
  const can = useCan();
  const canEdit = can(PERMS.orgManage);
  const crud = useCrud<JobProfile>('/job-profiles', 'jobProfiles');
  const deptName = useDepartmentNames();
  const [templates, setTemplates] = useState(false);
  const [generated, setGenerated] = useState<GenerateResult | null>(null);
  const [genBusy, setGenBusy] = useState(false);
  const ed = useEntityEditor({
    crud, id, blank, base: '/org/job-profiles', noun: 'Job profile', label: (j) => j.title,
    validate: (j) => (j.title.trim() ? null : 'Title is required'),
    normalize: (j) => ({ ...j, responsibilities: j.responsibilities ?? [], request_types: j.request_types ?? [] }),
  });
  const j = ed.draft;

  const applyTemplate = (t: JobTemplate) => {
    const { title_fa: _fa, id: _id, version: _v, updated_at: _u, ...rest } = t;
    void _fa; void _id; void _v; void _u;
    ed.setDraft({ ...blank, ...rest, department_id: j?.department_id ?? 0, responsibilities: rest.responsibilities ?? [], request_types: rest.request_types ?? [] });
    setTemplates(false);
  };
  const generate = async () => {
    if (!j) return;
    setGenBusy(true);
    try {
      const res = await post<GenerateResult>('/routing/generate', { job_profile_id: j.id });
      setGenerated(res);
      toast.success(`${res.rules?.length ?? 0} draft rule(s) created`);
    } catch (e) { toast.error(e); } finally { setGenBusy(false); }
  };

  return (
    <ConfigLayout
      title="Job profiles" sub="Job descriptions: what each role handles, its authority and when to escalate" basePath="/org/job-profiles"
      items={crud.items} loading={crud.loading} error={crud.error} onRetry={crud.reload}
      selected={id} canCreate={canEdit} newLabel="New job profile" emptyIcon={<IdCard />}
      search={(x) => `${x.title} ${x.description}`}
      renderItem={(x) => (
        <div className="grow">
          <div className="strong ellipsis" dir="auto">{x.title}</div>
          <div className="small muted ellipsis">{deptName(x.department_id)} · v{x.version}</div>
        </div>
      )}
    >
      {!id ? <div className="card"><Empty title="Select a job profile" icon={<IdCard />}>Job profiles feed routing: rules can target everyone holding a profile.</Empty></div>
        : !j ? <div className="card"><Empty title={crud.loading ? 'Loading…' : 'Job profile not found'} /></div>
        : (
          <Card title={<>{ed.isNew ? 'New job profile' : j.title || 'Job profile'}{!ed.isNew && <span className="badge info">v{j.version}</span>}</>} flush
            actions={<>
              {ed.isNew && canEdit && <button className="btn sm" onClick={() => setTemplates(true)}><LayoutTemplate />Start from template</button>}
              {!ed.isNew && can(PERMS.routingManage) && (
                <button className="btn sm" disabled={genBusy || ed.dirty} title={ed.dirty ? 'Save changes first' : 'Draft routing rules from this job description'} onClick={generate}>
                  <Sparkles />Generate routing rules
                </button>
              )}
            </>}>
            {generated && (
              <div className="alert info" style={{ margin: 16, marginBottom: 0 }}>
                <div className="grow">
                  {generated.rules?.length ?? 0} draft rule(s) generated ({generated.method === 'ai' ? 'by AI' : 'from template'}). Drafts do not route calls until approved.
                </div>
                <Link className="btn sm" to="/routing?tab=rules&status=draft">Review rules</Link>
              </div>
            )}
            <div className="card-body">
              {!ed.isNew && <div className="small muted" style={{ marginBottom: 12 }}>Version {j.version} · last saved {fmtDateTime(j.updated_at)}. Every save creates a new version.</div>}
              <JobProfileForm j={j} set={ed.set} disabled={!canEdit} />
            </div>
            <EditorFooter onSave={ed.save} onDelete={ed.remove} busy={ed.busy} canEdit={canEdit} dirty={ed.dirty} isNew={ed.isNew}
              extra={!ed.isNew && <RevisionsButton entity="job_profile" id={j.id} />} />
          </Card>
        )}
      <TemplatePicker open={templates} onClose={() => setTemplates(false)} onPick={applyTemplate} />
    </ConfigLayout>
  );
}
