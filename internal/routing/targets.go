package routing

import (
	"strconv"

	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"
)

func (e *Engine) queue(o *org, id int64) (domain.Action, string, string) {
	for _, q := range o.queues {
		if q.ID == id && q.Enabled {
			if d := o.depts[q.DepartmentID]; d != nil && !e.open(o, d) {
				return none, "", d.Name + " is closed"
			}
			return domain.Action{Type: domain.ActionQueue, Target: strconv.FormatInt(q.ID, 10)}, q.Name, "queue " + q.Name
		}
	}
	return none, "", "queue unavailable"
}

func (e *Engine) department(o *org, id int64) (domain.Action, string, string) {
	d := o.depts[id]
	if d == nil || !d.Enabled {
		return none, "", "department unavailable"
	}
	if !e.open(o, d) {
		return none, "", d.Name + " is closed"
	}
	for _, q := range o.queues {
		if q.DepartmentID == id && q.Enabled {
			return domain.Action{Type: domain.ActionQueue, Target: strconv.FormatInt(q.ID, 10)}, d.Name + " (" + q.Name + ")",
				d.Name + " queue " + q.Name
		}
	}
	return none, "", d.Name + " has no queue"
}

func (e *Engine) open(o *org, d *domain.Department) bool {
	s := o.sched[d.ScheduleID]
	return d.ScheduleID == 0 || s == nil || core.IsOpen(s, e.now())
}

func firstID(ids []int64) int64 {
	if len(ids) == 0 {
		return 0
	}
	return ids[0]
}

func joinNotes(n []string) string {
	out := ""
	for i, s := range n {
		if i > 0 {
			out += "; "
		}
		out += s
	}
	return out
}
