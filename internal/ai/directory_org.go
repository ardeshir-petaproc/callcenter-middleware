package ai

import (
	"context"
	"fmt"

	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
	"callcenter-middleware/internal/tickets"
)

// TopicNames lists the organization's request topics.
func (d *Directory) TopicNames(ctx context.Context) []string {
	topics, err := d.Store.Topics.List(ctx)
	if err != nil {
		return nil
	}
	var out []string
	for _, t := range topics {
		out = append(out, t.Name)
	}
	return out
}

// Route asks the routing engine where the caller should go and logs the
// decision with its explanation.
func (d *Directory) Route(ctx context.Context, spec core.AISpec, topic, request string) *domain.RoutingDecision {
	if d.Router == nil {
		return &domain.RoutingDecision{Reason: "routing engine not configured"}
	}
	req := domain.RouteRequest{Text: request, Topic: topic, Language: spec.Profile.Language,
		CallerNumber: spec.CallerNumber, CallID: spec.CallID}
	if c := core.FindContact(ctx, d.Store, spec.CallerNumber); c != nil {
		req.VIP = c.VIP
	}
	dec, err := d.Router.Decide(ctx, req)
	if err != nil {
		return &domain.RoutingDecision{Reason: "routing failed: " + err.Error()}
	}
	_ = d.Store.SaveDecision(ctx, dec)
	return dec
}

// SearchKnowledge returns passages the AI profile may use.
func (d *Directory) SearchKnowledge(ctx context.Context, p domain.AIProfile, query string) map[string]any {
	if d.KB == nil {
		return map[string]any{"results": []any{}, "note": "no knowledge base"}
	}
	tenant, _ := store.TenantOf(ctx)
	hits, err := d.KB.Search(ctx, tenant, query, domain.KBAccess{AI: true, DepartmentIDs: p.KBDepartmentIDs,
		Categories: p.KBCategories}, 3)
	if err != nil || len(hits) == 0 {
		return map[string]any{"results": []any{}, "note": "Nothing relevant found. Do not guess."}
	}
	var out []map[string]string
	for _, h := range hits {
		out = append(out, map[string]string{"source": h.Title, "text": h.Text})
	}
	return map[string]any{"results": out}
}

// CreateTicket opens a ticket linked to the call.
func (d *Directory) CreateTicket(ctx context.Context, spec core.AISpec, subject, description, priority, topic string) map[string]any {
	if d.Tickets == nil {
		return map[string]any{"ok": false, "error": "ticketing not configured"}
	}
	t := &domain.Ticket{Subject: subject, Description: description, Priority: priority, Category: topic, Source: "ai",
		RequesterPhone: spec.CallerNumber, RequesterName: spec.CallerName, CallIDs: []string{spec.CallID}}
	if c := core.FindContact(ctx, d.Store, spec.CallerNumber); c != nil {
		t.ContactID = c.ID
	}
	if topics, err := d.Store.Topics.List(ctx); err == nil {
		for _, tp := range topics {
			if tp.Name == topic {
				t.DepartmentID = tp.DepartmentID
			}
		}
	}
	if err := d.Tickets.Create(ctx, t, tickets.Actor{Name: "AI assistant"}); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true, "ticket_number": t.Number,
		"message": fmt.Sprintf("Ticket %s created; the team will follow up.", t.Number)}
}
