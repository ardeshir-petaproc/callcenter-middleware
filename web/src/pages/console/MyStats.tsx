import { useEffect } from 'react';
import { BarChart3 } from 'lucide-react';
import type { Summary } from '../../api/types';
import { get } from '../../api/client';
import { useApi } from '../../lib/hooks';
import { onLive } from '../../state/ws';
import { Card, Skeleton } from '../../components/ui/primitives';
import { fmtDuration, fmtLongDuration, fmtNum, fmtPct } from '../../lib/format';

export function MyStats() {
  const { data, reload } = useApi((s) => get<Summary>('/me/stats', undefined, s), []);
  useEffect(() => {
    let t: number | undefined;
    const off = onLive('call.ended', () => { clearTimeout(t); t = window.setTimeout(reload, 2500); });
    return () => { off(); clearTimeout(t); };
  }, [reload]);
  const items: [string, string][] = data ? [
    ['Calls', fmtNum(data.total)],
    ['Answered', fmtNum(data.answered)],
    ['Outbound', fmtNum(data.outbound)],
    ['Talk time', fmtLongDuration(data.total_talk_sec)],
    ['Avg talk', fmtDuration(data.avg_talk_sec)],
    ['SLA', data.answered ? fmtPct(data.sla_percent) : '—'],
  ] : [];
  return (
    <Card title="My day" icon={<BarChart3 />}>
      <div className="grid-3" style={{ gap: 12 }}>
        {data ? items.map(([k, v]) => (
          <div key={k}>
            <div className="small muted">{k}</div>
            <div className="strong num" style={{ fontSize: 20 }}>{v}</div>
          </div>
        )) : Array.from({ length: 6 }, (_, i) => <Skeleton key={i} h={36} />)}
      </div>
    </Card>
  );
}
