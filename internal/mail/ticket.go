package mail

import (
	"context"
	"strings"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/tickets"
)

// toTicket threads a customer reply onto its ticket (by [T-1234] in the
// subject or by In-Reply-To/References) or opens a new ticket. A reply only
// joins a ticket whose requester is the sender, and never a closed ticket
// (that starts a new one, as for chat), so nobody can write into someone
// else's ticket by quoting its number.
func (s *Service) toTicket(ctx context.Context, c *domain.EmailConnector, p *Parsed, text string) (int64, error) {
	customer := tickets.Actor{Name: firstNonEmpty(p.FromName, p.FromAddr)}
	var existing *domain.Ticket
	if m := ticketTag.FindStringSubmatch(p.Subject); m != nil {
		existing, _ = s.Store.TicketByNumber(ctx, m[1])
	}
	if existing == nil {
		if id, _ := s.Store.ThreadTicket(ctx, p.References); id > 0 {
			existing, _ = s.Store.GetTicket(ctx, id)
		}
	}
	if existing != nil && (existing.Status == domain.TicketClosed ||
		!strings.EqualFold(strings.TrimSpace(existing.RequesterEmail), p.FromAddr)) {
		existing = nil
	}
	if existing != nil {
		if _, err := s.Tickets.Comment(ctx, existing.ID, firstNonEmpty(text, "(empty message)"), false, customer); err != nil {
			return 0, err
		}
		if existing.Status == domain.TicketWaiting || existing.Status == domain.TicketResolved {
			status := domain.TicketInProgress
			if _, err := s.Tickets.Update(ctx, existing.ID, tickets.Patch{Status: &status}, customer); err != nil {
				return 0, err
			}
		}
		return existing.ID, nil
	}
	t := &domain.Ticket{Subject: firstNonEmpty(p.Subject, "Email from "+p.FromAddr), Description: text, Source: "email",
		Category: c.TicketCategory, DepartmentID: c.DepartmentID, RequesterEmail: p.FromAddr, RequesterName: p.FromName}
	if err := s.Tickets.Create(ctx, t, tickets.Actor{Name: "Email: " + c.Name}); err != nil {
		return 0, err
	}
	return t.ID, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
