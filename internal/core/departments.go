package core

import (
	"callcenter-middleware/internal/domain"
)

// departmentGate applies department policy (business hours, capacity) to a
// caller about to join q. It returns the fallback action and a reason when
// the caller must be diverted, or an empty reason to proceed.
func (c *Core) departmentGate(call *liveCall, q *domain.Queue) (domain.Action, string) {
	dept, ok := c.cache.departments[q.DepartmentID]
	if !ok || dept.TenantID != call.TenantID || !dept.Enabled {
		return domain.Action{}, ""
	}
	reason := ""
	if sch, ok := c.cache.schedules[dept.ScheduleID]; ok && dept.ScheduleID > 0 && !IsOpen(sch, c.now()) {
		reason = "closed"
	} else if dept.Capacity > 0 && c.departmentLoad(dept.ID, call) >= dept.Capacity {
		reason = "over_capacity"
	}
	if reason == "" {
		return domain.Action{}, ""
	}
	fb := dept.Fallback
	if fb.Type == domain.ActionNone {
		fb = domain.Action{Type: domain.ActionVoicemail}
	}
	// Never fall back into the same department (that would loop).
	if fb.Type == domain.ActionQueue {
		if tq := c.cache.findQueue(call.TenantID, fb.Target); tq == nil || tq.DepartmentID == dept.ID {
			fb = domain.Action{Type: domain.ActionVoicemail}
		}
	}
	return fb, reason
}

// departmentLoad counts calls currently waiting for or talking to a
// department, excluding the call being evaluated.
func (c *Core) departmentLoad(deptID int64, except *liveCall) int {
	n := 0
	for _, call := range c.calls {
		if call == except || call.queue == nil || call.queue.DepartmentID != deptID {
			continue
		}
		switch call.State {
		case domain.CallQueued, domain.CallRinging, domain.CallConnected, domain.CallHold:
			n++
		}
	}
	return n
}
