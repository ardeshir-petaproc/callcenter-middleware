package ai

import (
	"context"
	"strings"
	"testing"

	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/kb"
	"callcenter-middleware/internal/routing"
	"callcenter-middleware/internal/secure"
	"callcenter-middleware/internal/seed"
	"callcenter-middleware/internal/store"
	"callcenter-middleware/internal/tickets"
)

type allAvailable struct{}

func (allAvailable) Agent(id int64) (domain.AgentState, bool) {
	return domain.AgentState{UserID: id, TenantID: 1, Status: domain.AgentAvailable}, true
}

func TestDirectory_OrganizationTools(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := store.ForTenant(context.Background(), 1)
	if err := seed.Demo(ctx, st); err != nil {
		t.Fatal(err)
	}
	k := &kb.Service{Store: st, Box: secure.New("t"), Dir: t.TempDir()}
	d := &Directory{Store: st, Router: &routing.Engine{Store: st, Live: allAvailable{}}, KB: k,
		Tickets: &tickets.Service{Store: st}}
	spec := core.AISpec{TenantID: 1, CallID: "call-1", CallerNumber: "+15550001"}

	if names := d.TopicNames(ctx); len(names) != 4 {
		t.Fatalf("topics: %v", names)
	}
	dec := d.Route(ctx, spec, "Refund", "my order arrived broken, I want my money back")
	if dec.Action.Type != domain.ActionAgent || dec.RuleID == 0 {
		t.Fatalf("route: %+v", dec)
	}
	if logged, _ := st.Decisions(ctx, "call-1", 0, 10); len(logged) != 1 || logged[0].Reason == "" {
		t.Fatalf("decision log: %+v", logged)
	}

	_ = k.Save(ctx, &domain.KnowledgeSource{Title: "Returns", AIEnabled: true}, "", []byte("Broken items are replaced within 5 days."))
	_ = k.Save(ctx, &domain.KnowledgeSource{Title: "Internal", AIEnabled: false}, "", []byte("Broken items margin notes."))
	res := d.SearchKnowledge(ctx, domain.AIProfile{}, "broken items")
	if hits := res["results"].([]map[string]string); len(hits) != 1 || hits[0]["source"] != "Returns" {
		t.Fatalf("knowledge: %+v", res)
	}

	out := d.CreateTicket(ctx, spec, "Broken item", "Order arrived broken", "high", "Refund")
	if out["ok"] != true || out["ticket_number"] == "" {
		t.Fatalf("ticket: %+v", out)
	}
	tk, err := st.TicketByNumber(ctx, out["ticket_number"].(string))
	if err != nil || tk.Source != "ai" || tk.DepartmentID == 0 || len(tk.CallIDs) != 1 {
		t.Fatalf("stored ticket: %+v %v", tk, err)
	}
	// Callers can only look up and update their own tickets.
	if st := d.TicketStatus(ctx, spec, strings.TrimPrefix(tk.Number, "T-")); st["found"] != true || st["status"] != "new" {
		t.Fatalf("status: %+v", st)
	}
	stranger := core.AISpec{TenantID: 1, CallID: "call-2", CallerNumber: "+15559999"}
	if st := d.TicketStatus(ctx, stranger, tk.Number); st["found"] != false {
		t.Fatalf("stranger must not see the ticket: %+v", st)
	}
	_ = st.SaveCall(ctx, &domain.Call{ID: "call-3", Direction: "inbound", State: domain.CallAI, Tags: []string{}})
	if up := d.UpdateTicket(ctx, core.AISpec{TenantID: 1, CallID: "call-3", CallerNumber: "+1 555 0001"}, tk.Number,
		"Photos sent by email"); up["ok"] != true {
		t.Fatalf("update: %+v", up)
	}
	comments, _ := st.TicketComments(ctx, tk.ID)
	if len(comments) != 1 || !strings.Contains(comments[0].Body, "Photos sent") {
		t.Fatalf("comment: %+v", comments)
	}
	if tk, _ = st.GetTicket(ctx, tk.ID); len(tk.CallIDs) != 2 {
		t.Fatalf("call not linked: %+v", tk.CallIDs)
	}
}
