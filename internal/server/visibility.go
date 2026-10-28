package server

import (
	"net/http"

	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"
)

// visibleSnapshot returns the caller's tenant snapshot. Without calls.all,
// only the caller's own calls (talking or ringing) are included.
func (s *Server) visibleSnapshot(r *http.Request) core.Snapshot {
	snap := s.d.Core.Snapshot(tenantOf(r))
	if can(r, domain.PermCallsAll) {
		return snap
	}
	me := claimsOf(r).UserID
	mine := []core.CallView{}
	for _, c := range snap.Calls {
		if ownsCall(c, me) {
			mine = append(mine, c)
		}
	}
	snap.Calls = mine
	return snap
}

func ownsCall(c core.CallView, me int64) bool {
	if c.AgentID == me {
		return true
	}
	for _, id := range c.RingingAgents {
		if id == me {
			return true
		}
	}
	return false
}

// eventFilter enforces tenant isolation and per-user visibility on the
// realtime stream.
type eventFilter struct {
	tenant  int64
	me      int64
	all     bool
	myCalls map[string]bool
	tickets *domain.TicketVisibility
}

func (f *eventFilter) seesTicket(t *domain.Ticket) bool {
	v := f.tickets
	if v == nil || t.AssigneeID == v.UserID || t.CreatedBy == v.UserID {
		return true
	}
	for _, d := range v.DepartmentIDs {
		if d == t.DepartmentID {
			return true
		}
	}
	return false
}

func newEventFilter(r *http.Request) *eventFilter {
	return &eventFilter{tenant: tenantOf(r), me: claimsOf(r).UserID, all: can(r, domain.PermCallsAll),
		myCalls: map[string]bool{}, tickets: visibility(r)}
}

func (f *eventFilter) allow(ev domain.Event) bool {
	if ev.TenantID != f.tenant {
		return false
	}
	if f.all {
		return true
	}
	switch data := ev.Data.(type) {
	case core.CallView:
		if ownsCall(data, f.me) {
			f.myCalls[data.ID] = true
			return true
		}
		// Keep delivering updates for calls that were ours (e.g. call.ended).
		return f.myCalls[data.ID]
	case domain.TranscriptEntry:
		return f.myCalls[data.CallID]
	case *domain.Ticket:
		return f.seesTicket(data)
	case map[string]any:
		if uid, ok := data["user_id"].(int64); ok {
			return uid == f.me
		}
	}
	return true
}
