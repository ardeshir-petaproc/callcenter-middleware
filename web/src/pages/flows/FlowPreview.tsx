import { Clock, Hash, Moon, PhoneIncoming, Volume2, FastForward } from 'lucide-react';
import type { IVRFlow } from '../../api/types';
import { useDescribeAction } from '../../components/ActionEditor';
import { useLookup } from '../../state/lookups';
import { KEYS } from './KeypadEditor';

/** Read-only tree of what a caller experiences in this flow. */
export function FlowPreview({ flow }: { flow: IVRFlow }) {
  const describe = useDescribeAction();
  const schedules = useLookup('schedules');
  const schedule = schedules.find((s) => s.id === flow.schedule_id);
  const keys = KEYS.filter((k) => flow.options?.[k]?.type);
  const direct = !!flow.direct_action?.type;
  return (
    <div className="tree">
      <span className="node root"><PhoneIncoming />{flow.name || 'Untitled flow'}{flow.dnis ? ` · DNIS ${flow.dnis}` : ''}</span>
      <ul>
        {schedule && (
          <li>
            <span className="node"><Moon />Outside “{schedule.name}” hours → {describe(flow.after_hours_action)}</span>
          </li>
        )}
        {flow.greeting_media && <li><span className="node"><Volume2 />Play {flow.greeting_media}</span></li>}
        {direct ? (
          <li><span className="node"><FastForward />Skip menu → {describe(flow.direct_action)}</span></li>
        ) : (
          <>
            <li>
              <span className="node"><Hash />Menu ({keys.length} option{keys.length === 1 ? '' : 's'})</span>
              <ul>
                {keys.length === 0 && <li><span className="node muted">No keys configured</span></li>}
                {keys.map((k) => <li key={k}><span className="node"><b>{k}</b>→ {describe(flow.options[k])}</span></li>)}
                <li><span className="node muted">Invalid key → {flow.invalid_media || 'replay'} (max {flow.max_retries} retries)</span></li>
              </ul>
            </li>
            <li><span className="node"><Clock />No input after {flow.timeout_sec}s → {describe(flow.timeout_action)}</span></li>
          </>
        )}
      </ul>
    </div>
  );
}
