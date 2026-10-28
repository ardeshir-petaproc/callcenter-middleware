package core_test

import (
	"testing"

	"callcenter-middleware/internal/domain"
)

func TestDepartment_ClosedAndCapacityUseFallback(t *testing.T) {
	h := newHarness(t, nil)
	a := h.agent("a", "1001")
	closed := &domain.Schedule{Name: "never", Weekly: []domain.TimeRange{{Day: 0, Start: "00:00", End: "00:01"}}}
	_ = h.st.Schedules.Create(tctx(), closed)
	dept := &domain.Department{Name: "Sales", Enabled: true, ScheduleID: closed.ID,
		Fallback: domain.Action{Type: domain.ActionCallback}}
	_ = h.st.Departments.Create(tctx(), dept)
	q := h.queue(domain.Queue{Name: "Sales", DepartmentID: dept.ID}, a)
	setupQueueFlow(h, q)
	_ = h.core.SetAgentStatus(a.ID, domain.AgentAvailable, "")

	h.incoming("c1", "555", "200")
	h.dtmf("c1", "1")
	if h.sw.has("originate PJSIP/1001") {
		t.Fatal("closed department must not ring agents")
	}
	cbs, _ := h.st.ListCallbacks(tctx(), "", 10)
	if len(cbs) != 1 {
		t.Fatalf("expected department fallback (callback), got %+v", cbs)
	}

	// Open the department but cap it at one concurrent call.
	dept.ScheduleID, dept.Capacity = 0, 1
	_ = h.st.Departments.Update(tctx(), dept)
	h.reload()
	h.incoming("c2", "556", "200")
	h.dtmf("c2", "1")
	h.incoming("c3", "557", "200")
	h.dtmf("c3", "1")
	if cbs, _ = h.st.ListCallbacks(tctx(), "", 10); len(cbs) != 2 {
		t.Fatalf("second caller should overflow by capacity, got %d callbacks", len(cbs))
	}
}

func TestTenantRouting_NumberSelectsTenantAndFlow(t *testing.T) {
	h := newHarness(t, nil)
	other := &domain.Tenant{Slug: "other", Name: "Other"}
	_ = h.st.CreateTenant(tctx(), other)
	octx := ctxFor(other.ID)
	q := &domain.Queue{Name: "Other queue", Enabled: true}
	_ = h.st.Queues.Create(octx, q)
	flow := &domain.IVRFlow{Name: "other", Enabled: true, DirectAction: domain.Action{Type: domain.ActionQueue, Target: itoa(q.ID)}}
	_ = h.st.Flows.Create(octx, flow)
	_ = h.st.Numbers.Create(octx, &domain.PhoneNumber{Number: "+1 555 0300", FlowID: flow.ID, Enabled: true})
	// A default-tenant queue whose id the other tenant's flow must not reach.
	mine := h.queue(domain.Queue{Name: "Mine"})
	h.reload()

	h.incoming("c1", "999", "15550300")
	snap := h.core.Snapshot(other.ID)
	if len(snap.Calls) != 1 || snap.Calls[0].QueueID != q.ID {
		t.Fatalf("call should be queued in the other tenant, got %+v", snap.Calls)
	}
	if len(h.core.Snapshot(1).Calls) != 0 {
		t.Fatal("default tenant must not see the other tenant's call")
	}
	// Transfers can never target another tenant's queue.
	_ = h.core.Transfer(snap.Calls[0].ID, domain.Action{Type: domain.ActionQueue, Target: itoa(mine.ID)})
	if v, _ := h.core.LiveCall(snap.Calls[0].ID); v.QueueID == mine.ID {
		t.Fatal("cross-tenant transfer must be refused")
	}
}
