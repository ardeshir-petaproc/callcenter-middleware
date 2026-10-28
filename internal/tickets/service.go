// Package tickets implements the ticket workflow: creation with SLA
// deadlines, assignment, status changes, replies vs internal notes, change
// history, call links and assignee notifications.
package tickets

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/joow"
	"callcenter-middleware/internal/store"
)

// Event type published on every ticket change.
const EvTicketUpdated = "ticket.updated"

// ErrInvalid is returned for invalid ticket data.
var ErrInvalid = errors.New("invalid ticket")

// Actor performs a change (a user, or the AI / system with ID 0).
type Actor struct {
	ID   int64
	Name string
}

// Service is the ticket workflow.
type Service struct {
	Store *store.Store
	Pub   domain.Publisher
	Joow  joow.Provider
	Now   func() time.Time
	// OnReply delivers staff replies to the customer (e.g. by email). It
	// runs in the background with a tenant-scoped context.
	OnReply func(ctx context.Context, t *domain.Ticket, c *domain.TicketComment)
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

func validPriority(p string) bool {
	switch p {
	case domain.PriorityLow, domain.PriorityNormal, domain.PriorityHigh, domain.PriorityUrgent:
		return true
	}
	return false
}

func validStatus(st string) bool {
	for _, s := range domain.TicketStatuses {
		if s == st {
			return true
		}
	}
	return false
}

// Create opens a ticket, applying the tenant's SLA policy for its priority.
func (s *Service) Create(ctx context.Context, t *domain.Ticket, by Actor) error {
	t.Subject = strings.TrimSpace(t.Subject)
	if t.Subject == "" {
		return fmt.Errorf("%w: subject is required", ErrInvalid)
	}
	if t.Priority == "" {
		t.Priority = domain.PriorityNormal
	}
	if !validPriority(t.Priority) {
		return fmt.Errorf("%w: unknown priority", ErrInvalid)
	}
	if t.Source == "" {
		t.Source = "manual"
	}
	t.Status = domain.TicketNew
	if t.AssigneeID > 0 {
		t.Status = domain.TicketAssigned
	}
	t.CreatedBy = by.ID
	s.applySLA(ctx, t, s.now())
	if err := s.Store.CreateTicket(ctx, t); err != nil {
		return err
	}
	s.event(ctx, t, "created", "", t.Number, by)
	if t.AssigneeID > 0 {
		s.notifyAssignee(ctx, t)
	}
	return nil
}

// applySLA sets response and resolution deadlines from the tenant policy.
func (s *Service) applySLA(ctx context.Context, t *domain.Ticket, from time.Time) {
	policies, err := s.Store.SLAPolicies.List(ctx)
	if err != nil {
		return
	}
	for _, p := range policies {
		if p.Priority != t.Priority {
			continue
		}
		if p.FirstResponseMinutes > 0 && t.FirstResponse == nil {
			d := from.Add(time.Duration(p.FirstResponseMinutes) * time.Minute).UTC()
			t.ResponseDue = &d
		}
		if p.ResolutionMinutes > 0 {
			d := from.Add(time.Duration(p.ResolutionMinutes) * time.Minute).UTC()
			t.ResolutionDue = &d
		}
	}
}

func (s *Service) event(ctx context.Context, t *domain.Ticket, typ, from, to string, by Actor) {
	_ = s.Store.AddTicketEvent(ctx, t.TenantID, domain.TicketEvent{TicketID: t.ID, Type: typ, From: from, To: to,
		ActorID: by.ID, ActorName: by.Name})
	if s.Pub != nil {
		s.Pub.Publish(domain.Event{TenantID: t.TenantID, Type: EvTicketUpdated, Data: t})
	}
}

func (s *Service) notifyAssignee(ctx context.Context, t *domain.Ticket) {
	if s.Joow == nil {
		return
	}
	u, err := s.Store.GetUser(ctx, t.AssigneeID)
	if err != nil || u.JoowAccountID == "" {
		return
	}
	_ = s.Joow.Notify(ctx, u.JoowAccountID, joow.Notification{Kind: "ticket_assigned",
		Title: t.Number + " · " + t.Subject, Body: "A ticket was assigned to you (" + t.Priority + ").",
		Data: map[string]any{"ticket_id": t.ID}})
}
