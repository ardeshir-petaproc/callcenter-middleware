package tickets

import (
	"context"
	"fmt"
	"strings"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
)

// Patch holds optional field changes.
type Patch struct {
	Subject      *string   `json:"subject"`
	Description  *string   `json:"description"`
	Category     *string   `json:"category"`
	Priority     *string   `json:"priority"`
	Status       *string   `json:"status"`
	DepartmentID *int64    `json:"department_id"`
	AssigneeID   *int64    `json:"assignee_id"`
	ContactID    *int64    `json:"contact_id"`
	Tags         *[]string `json:"tags"`
}

// Update applies a patch, recording every change in the history.
func (s *Service) Update(ctx context.Context, id int64, p Patch, by Actor) (*domain.Ticket, error) {
	t, err := s.Store.GetTicket(ctx, id)
	if err != nil {
		return nil, err
	}
	set := func(field string, cur *string, next *string) {
		if next != nil && *next != *cur {
			s.event(ctx, t, field, *cur, *next, by)
			*cur = *next
		}
	}
	setID := func(field string, cur *int64, next *int64) {
		if next != nil && *next != *cur {
			s.event(ctx, t, field, fmt.Sprint(*cur), fmt.Sprint(*next), by)
			*cur = *next
		}
	}
	if p.Priority != nil && !validPriority(*p.Priority) {
		return nil, fmt.Errorf("%w: unknown priority", ErrInvalid)
	}
	if p.Status != nil && !validStatus(*p.Status) {
		return nil, fmt.Errorf("%w: unknown status", ErrInvalid)
	}
	if p.Subject != nil && strings.TrimSpace(*p.Subject) == "" {
		return nil, fmt.Errorf("%w: subject is required", ErrInvalid)
	}
	prevAssignee, prevPriority := t.AssigneeID, t.Priority
	set("subject", &t.Subject, p.Subject)
	set("description", &t.Description, p.Description)
	set("category", &t.Category, p.Category)
	set("priority", &t.Priority, p.Priority)
	setID("department", &t.DepartmentID, p.DepartmentID)
	setID("assignee", &t.AssigneeID, p.AssigneeID)
	setID("contact", &t.ContactID, p.ContactID)
	if p.Tags != nil {
		t.Tags = *p.Tags
	}
	if t.AssigneeID != prevAssignee && t.AssigneeID > 0 && t.Status == domain.TicketNew && p.Status == nil {
		next := domain.TicketAssigned
		p.Status = &next
	}
	prevStatus := t.Status
	set("status", &t.Status, p.Status)
	s.stamp(t, prevStatus)
	if t.Priority != prevPriority {
		s.applySLA(ctx, t, t.CreatedAt)
	}
	if err := s.Store.UpdateTicket(ctx, t); err != nil {
		return nil, err
	}
	if t.AssigneeID != prevAssignee && t.AssigneeID > 0 {
		s.notifyAssignee(ctx, t)
	}
	return t, nil
}

// stamp maintains resolved/closed timestamps across status transitions.
func (s *Service) stamp(t *domain.Ticket, prev string) {
	now := s.now().UTC()
	switch t.Status {
	case domain.TicketResolved:
		if t.ResolvedAt == nil {
			t.ResolvedAt = &now
		}
	case domain.TicketClosed:
		if t.ResolvedAt == nil {
			t.ResolvedAt = &now
		}
		if t.ClosedAt == nil {
			t.ClosedAt = &now
		}
	default:
		if prev == domain.TicketResolved || prev == domain.TicketClosed { // reopened
			t.ResolvedAt, t.ClosedAt = nil, nil
		}
	}
}

// Comment adds a customer-visible reply or an internal note. The first
// public reply by staff counts as the first response for SLA purposes.
func (s *Service) Comment(ctx context.Context, id int64, body string, internal bool, by Actor) (*domain.TicketComment, error) {
	t, err := s.Store.GetTicket(ctx, id)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(body) == "" {
		return nil, fmt.Errorf("%w: comment body is required", ErrInvalid)
	}
	c := &domain.TicketComment{TicketID: id, AuthorID: by.ID, AuthorName: by.Name, Body: body, Internal: internal}
	if err := s.Store.AddTicketComment(ctx, t.TenantID, c); err != nil {
		return nil, err
	}
	if !internal && by.ID > 0 && t.FirstResponse == nil {
		now := s.now().UTC()
		t.FirstResponse = &now
		_ = s.Store.UpdateTicket(ctx, t)
	}
	kind := "reply"
	if internal {
		kind = "note"
	} else if by.ID > 0 && s.OnReply != nil {
		go s.OnReply(store.ForTenant(context.Background(), t.TenantID), t, c)
	}
	s.event(ctx, t, kind, "", "", by)
	return c, nil
}

// LinkCall attaches a call to the ticket.
func (s *Service) LinkCall(ctx context.Context, id int64, callID string, by Actor) (*domain.Ticket, error) {
	t, err := s.Store.GetTicket(ctx, id)
	if err != nil {
		return nil, err
	}
	if _, err := s.Store.GetCall(ctx, callID); err != nil {
		return nil, err
	}
	for _, c := range t.CallIDs {
		if c == callID {
			return t, nil
		}
	}
	t.CallIDs = append(t.CallIDs, callID)
	if err := s.Store.UpdateTicket(ctx, t); err != nil {
		return nil, err
	}
	s.event(ctx, t, "call_linked", "", callID, by)
	return t, nil
}
