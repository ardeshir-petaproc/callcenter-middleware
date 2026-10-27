package ai

import (
	"context"
	"time"

	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/store"
)

// Directory answers tool lookups from the store.
type Directory struct {
	Store *store.Store
}

// QueueNames lists enabled queues the AI may transfer to.
func (d *Directory) QueueNames(ctx context.Context) []string {
	queues, err := d.Store.Queues.List(ctx)
	if err != nil {
		return nil
	}
	var out []string
	for _, q := range queues {
		if q.Enabled {
			out = append(out, q.Name)
		}
	}
	return out
}

// LookupContact returns the CRM record and recent calls for a number.
func (d *Directory) LookupContact(ctx context.Context, phone string) map[string]any {
	res := map[string]any{"found": false}
	if c := core.FindContact(ctx, d.Store, phone); c != nil {
		res = map[string]any{"found": true, "name": c.Name, "company": c.Company, "email": c.Email,
			"notes": c.Notes, "vip": c.VIP, "tags": c.Tags}
	}
	calls, err := d.Store.RecentCallsByNumber(ctx, core.NormalizePhone(phone), 5)
	if err == nil {
		var recent []map[string]any
		for _, call := range calls {
			recent = append(recent, map[string]any{
				"date": call.StartedAt.Format("2006-01-02"), "outcome": call.Outcome,
				"disposition": call.Disposition, "summary": call.Summary, "notes": call.Notes,
			})
		}
		res["recent_calls"] = recent
	}
	return res
}

// BusinessHours reports which schedules are open now.
func (d *Directory) BusinessHours(ctx context.Context) map[string]any {
	schedules, err := d.Store.Schedules.List(ctx)
	if err != nil || len(schedules) == 0 {
		return map[string]any{"open": true, "note": "no schedule configured"}
	}
	now := time.Now()
	out := map[string]any{}
	anyOpen := false
	for i := range schedules {
		open := core.IsOpen(&schedules[i], now)
		anyOpen = anyOpen || open
		out[schedules[i].Name] = map[string]any{"open": open, "hours": schedules[i].Weekly}
	}
	out["open"] = anyOpen
	return out
}
