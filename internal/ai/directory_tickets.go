package ai

import (
	"context"
	"strings"

	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/tickets"
)

// callerTicket loads a ticket by number, but only if it belongs to the
// caller (same phone number or CRM contact), so callers cannot read or
// change other customers' tickets.
func (d *Directory) callerTicket(ctx context.Context, spec core.AISpec, number string) (*domain.Ticket, string) {
	number = strings.ToUpper(strings.TrimSpace(number))
	if number != "" && !strings.HasPrefix(number, "T-") {
		number = "T-" + strings.TrimPrefix(number, "T")
	}
	t, err := d.Store.TicketByNumber(ctx, number)
	if err != nil {
		return nil, "No ticket with that number."
	}
	caller := core.NormalizePhone(spec.CallerNumber)
	owned := caller != "" && core.NormalizePhone(t.RequesterPhone) == caller
	if c := core.FindContact(ctx, d.Store, spec.CallerNumber); c != nil && t.ContactID == c.ID {
		owned = true
	}
	if !owned {
		return nil, "That ticket is not linked to the number the caller is calling from; offer to connect them to an employee."
	}
	return t, ""
}

// TicketStatus tells a caller where their ticket stands.
func (d *Directory) TicketStatus(ctx context.Context, spec core.AISpec, number string) map[string]any {
	t, why := d.callerTicket(ctx, spec, number)
	if t == nil {
		return map[string]any{"found": false, "message": why}
	}
	return map[string]any{"found": true, "number": t.Number, "subject": t.Subject, "status": t.Status,
		"priority": t.Priority, "updated_at": t.UpdatedAt.Format("2006-01-02 15:04")}
}

// UpdateTicket adds the caller's new information to their ticket and puts
// it back in the team's hands.
func (d *Directory) UpdateTicket(ctx context.Context, spec core.AISpec, number, message string) map[string]any {
	if d.Tickets == nil {
		return map[string]any{"ok": false, "error": "ticketing not configured"}
	}
	t, why := d.callerTicket(ctx, spec, number)
	if t == nil {
		return map[string]any{"ok": false, "message": why}
	}
	who := tickets.Actor{Name: "Caller via AI assistant"}
	if _, err := d.Tickets.Comment(ctx, t.ID, "[Phone, call "+spec.CallID+"] "+message, false, who); err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	if t.Status == domain.TicketWaiting || !t.Open() {
		status := domain.TicketInProgress
		_, _ = d.Tickets.Update(ctx, t.ID, tickets.Patch{Status: &status}, who)
	}
	_, _ = d.Tickets.LinkCall(ctx, t.ID, spec.CallID, who)
	return map[string]any{"ok": true, "message": "Added to ticket " + t.Number + "; the team will see it."}
}
