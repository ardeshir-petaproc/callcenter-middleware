package integration

import (
	"testing"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/sim"
)

func TestTickets_WorkflowSLAAndVisibility(t *testing.T) {
	s := newStack(t, 40)
	admin := s.login("admin", "admin123")
	var depts []domain.Department
	s.call("GET", "/departments", admin, nil, &depts)
	sales, finance := depts[0], depts[2]
	s.call("POST", "/sla-policies", admin, domain.SLAPolicy{Priority: "high", FirstResponseMinutes: 30, ResolutionMinutes: 240}, nil)

	var tk domain.Ticket
	body := map[string]any{"subject": "Refund for order 42", "priority": "high", "category": "refund", "department_id": finance.ID}
	if code := s.call("POST", "/tickets", admin, body, &tk); code != 201 || tk.Number == "" || tk.ResponseDue == nil {
		t.Fatalf("create: %d %+v", code, tk)
	}
	// Alice (sales) cannot see a finance ticket; Dave (finance) can.
	alice, dave := s.login("alice", "agent123"), s.login("dave", "agent123")
	if code := s.call("GET", "/tickets/"+itoa(tk.ID), alice, nil, nil); code != 403 {
		t.Fatalf("alice must not see finance ticket, got %d", code)
	}
	var list struct {
		Total int `json:"total"`
	}
	s.call("GET", "/tickets?q="+tk.Number, dave, nil, &list)
	if list.Total != 1 {
		t.Fatalf("dave should see his department's ticket, got %d", list.Total)
	}
	s.call("GET", "/tickets?q="+tk.Number, alice, nil, &list)
	if list.Total != 0 {
		t.Fatalf("alice's list must not include it, got %d", list.Total)
	}
	// Internal notes do not count as first response; public replies do.
	s.call("POST", "/tickets/"+itoa(tk.ID)+"/comments", dave, map[string]any{"body": "checking payment", "internal": true}, nil)
	var detail struct {
		Ticket   domain.Ticket          `json:"ticket"`
		Comments []domain.TicketComment `json:"comments"`
		History  []domain.TicketEvent   `json:"history"`
	}
	s.call("GET", "/tickets/"+itoa(tk.ID), dave, nil, &detail)
	if detail.Ticket.FirstResponse != nil {
		t.Fatal("internal note must not count as first response")
	}
	s.call("POST", "/tickets/"+itoa(tk.ID)+"/comments", dave, map[string]any{"body": "Refund issued."}, nil)
	var users []domain.User
	s.call("GET", "/users", admin, nil, &users)
	var daveID int64
	for _, u := range users {
		if u.Username == "dave" {
			daveID = u.ID
		}
	}
	s.call("PATCH", "/tickets/"+itoa(tk.ID), dave, map[string]any{"assignee_id": daveID, "status": "resolved"}, nil)
	s.call("GET", "/tickets/"+itoa(tk.ID), dave, nil, &detail)
	if detail.Ticket.FirstResponse == nil || detail.Ticket.ResolvedAt == nil || detail.Ticket.Status != "resolved" {
		t.Fatalf("unexpected ticket %+v", detail.Ticket)
	}
	if len(detail.Comments) != 2 || len(detail.History) < 4 {
		t.Fatalf("expected comments and history, got %d / %d", len(detail.Comments), len(detail.History))
	}
	if code := s.call("PATCH", "/tickets/"+itoa(tk.ID), dave, map[string]any{"status": "bogus"}, nil); code != 400 {
		t.Fatalf("invalid status must 400, got %d", code)
	}
	// Other organizations never see this ticket.
	shop := s.loginTenant("admin", "admin123", "parsian")
	if code := s.call("GET", "/tickets/"+itoa(tk.ID), shop, nil, nil); code != 404 {
		t.Fatalf("cross-tenant ticket must 404, got %d", code)
	}
	_ = sales
}

func TestTickets_FromCall(t *testing.T) {
	s := newStack(t, 40)
	alice := s.login("alice", "agent123")
	s.call("POST", "/me/status", alice, map[string]string{"status": "available"}, nil)
	s.sim.InjectCall(sim.SimCall{Caller: "+15550100", DNIS: "100", Digits: "1", TalkSec: 5})
	var calls struct {
		Items []domain.Call `json:"items"`
	}
	waitFor(t, 5*time.Second, "answered call", func() bool {
		s.call("GET", "/calls?outcome=answered&q=%2B15550100", alice, nil, &calls)
		return len(calls.Items) == 1
	})
	var tk domain.Ticket
	if code := s.call("POST", "/calls/"+calls.Items[0].ID+"/ticket", alice, map[string]string{"category": "sales"}, &tk); code != 201 {
		t.Fatalf("ticket from call: %d", code)
	}
	if tk.Source != "call" || len(tk.CallIDs) != 1 || tk.ContactID == 0 || tk.RequesterName != "Maria Rossi" || tk.AssigneeID == 0 {
		t.Fatalf("unexpected ticket %+v", tk)
	}
	var found struct {
		Total int `json:"total"`
	}
	s.call("GET", "/tickets?call_id="+calls.Items[0].ID, alice, nil, &found)
	if found.Total != 1 {
		t.Fatal("ticket should be findable by call")
	}
}
