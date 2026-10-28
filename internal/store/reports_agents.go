package store

import (
	"context"
	"time"

	"callcenter-middleware/internal/domain"
)

// AgentTime is the number of seconds an agent spent in each status.
type AgentTime struct {
	UserID  int64          `json:"user_id"`
	Seconds map[string]int `json:"seconds"`
}

// ReportAgentTime replays agent status transitions between from and to and
// sums the time spent in each status. The last status before from is used
// as the starting state so long-running statuses are counted correctly.
func (s *Store) ReportAgentTime(ctx context.Context, from, to time.Time) ([]AgentTime, error) {
	cond, targs, err := scoped(ctx, "tenant_id")
	if err != nil {
		return nil, err
	}
	args := append([]any{ms(to), ms(from), ms(from)}, targs...)
	rows, err := s.db.QueryContext(ctx, `SELECT user_id, status, at FROM agent_events
		WHERE at < ? AND (at >= ? OR id IN (SELECT MAX(id) FROM agent_events WHERE at < ? GROUP BY user_id))`+cond+`
		ORDER BY user_id, at, id`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type cursor struct {
		status string
		since  time.Time
	}
	totals := map[int64]map[string]int{}
	cur := map[int64]*cursor{}
	order := []int64{}
	for rows.Next() {
		var uid int64
		var status string
		var at int64
		if err := rows.Scan(&uid, &status, &at); err != nil {
			return nil, err
		}
		t := fromMS(at)
		if t.Before(from) {
			t = from
		}
		if _, ok := totals[uid]; !ok {
			totals[uid] = map[string]int{}
			order = append(order, uid)
		}
		if c := cur[uid]; c != nil {
			totals[uid][c.status] += int(t.Sub(c.since).Seconds())
		}
		cur[uid] = &cursor{status: status, since: t}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	end := to
	if now := time.Now(); now.Before(end) {
		end = now
	}
	out := make([]AgentTime, 0, len(order))
	for _, uid := range order {
		if c := cur[uid]; c != nil && c.since.Before(end) {
			totals[uid][c.status] += int(end.Sub(c.since).Seconds())
		}
		delete(totals[uid], string(domain.AgentOffline))
		out = append(out, AgentTime{UserID: uid, Seconds: totals[uid]})
	}
	return out, nil
}
