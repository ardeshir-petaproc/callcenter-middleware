package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/tickets"

	"github.com/go-chi/chi/v5"
)

func (s *Server) ticketRoutes(r chi.Router) {
	r.Get("/tickets", handler(s.listTickets).ServeHTTP)
	r.Post("/tickets", handler(s.createTicket).ServeHTTP)
	r.With(requirePerm(domain.PermReportsView)).Get("/tickets/stats", handler(s.ticketStats).ServeHTTP)
	r.Get("/tickets/{id}", handler(s.getTicket).ServeHTTP)
	r.Patch("/tickets/{id}", handler(s.patchTicket).ServeHTTP)
	r.Post("/tickets/{id}/comments", handler(s.commentTicket).ServeHTTP)
	r.Post("/tickets/{id}/link-call", handler(s.linkTicketCall).ServeHTTP)
	r.Post("/calls/{id}/ticket", handler(s.ticketFromCall).ServeHTTP)
	mountCRUD(r, "/sla-policies", s, s.d.Store.SLAPolicies, crudOpts[domain.SLAPolicy]{entity: "sla_policy",
		prepare: prepareSLA})
}

func actor(r *http.Request) tickets.Actor {
	p := principalOf(r)
	return tickets.Actor{ID: p.user.ID, Name: p.user.DisplayName}
}

// visibility limits tickets for staff without tickets.all to their own and
// their departments' tickets.
func visibility(r *http.Request) *domain.TicketVisibility {
	if can(r, domain.PermTicketsAll) {
		return nil
	}
	u := principalOf(r).user
	return &domain.TicketVisibility{UserID: u.ID, DepartmentIDs: u.DepartmentIDs}
}

func canSeeTicket(r *http.Request, t *domain.Ticket) bool {
	v := visibility(r)
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

func ticketErr(err error) error {
	if errors.Is(err, tickets.ErrInvalid) {
		return bad(strings.TrimPrefix(err.Error(), tickets.ErrInvalid.Error()+": "))
	}
	return err
}

func (s *Server) listTickets(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	f := domain.TicketFilter{Status: q.Get("status"), Priority: q.Get("priority"), Category: q.Get("category"),
		DepartmentID: qInt64(r, "department_id"), AssigneeID: qInt64(r, "assignee_id"), Search: q.Get("q"),
		CallID: q.Get("call_id"), VisibleTo: visibility(r), Limit: qInt(r, "limit", 50), Offset: qInt(r, "offset", 0)}
	if q.Get("mine") == "1" {
		f.AssigneeID = claimsOf(r).UserID
	}
	if q.Get("overdue") == "1" {
		now := time.Now()
		f.OverdueAt = &now
	}
	items, total, err := s.d.Store.ListTickets(r.Context(), f)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
	return nil
}

func (s *Server) checkTicketRefs(r *http.Request, t *domain.Ticket) error {
	if err := s.exists(r, "department", t.DepartmentID); err != nil {
		return err
	}
	if err := s.exists(r, "user", t.AssigneeID); err != nil {
		return err
	}
	if t.ContactID > 0 {
		if _, err := s.d.Store.Contacts.Get(r.Context(), t.ContactID); err != nil {
			return bad("contact does not exist in this organization")
		}
	}
	return nil
}

func (s *Server) createTicket(w http.ResponseWriter, r *http.Request) error {
	var t domain.Ticket
	if err := decode(r, &t); err != nil {
		return err
	}
	t.ID, t.TenantID, t.CallIDs = 0, 0, nil
	if err := s.checkTicketRefs(r, &t); err != nil {
		return err
	}
	if err := s.d.Tickets.Create(r.Context(), &t, actor(r)); err != nil {
		return ticketErr(err)
	}
	writeJSON(w, http.StatusCreated, t)
	return nil
}

func (s *Server) loadTicket(r *http.Request) (*domain.Ticket, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	t, err := s.d.Store.GetTicket(r.Context(), id)
	if err != nil {
		return nil, err
	}
	if !canSeeTicket(r, t) {
		return nil, errForbidden
	}
	return t, nil
}

func (s *Server) getTicket(w http.ResponseWriter, r *http.Request) error {
	t, err := s.loadTicket(r)
	if err != nil {
		return err
	}
	comments, err := s.d.Store.TicketComments(r.Context(), t.ID)
	if err != nil {
		return err
	}
	history, err := s.d.Store.TicketEvents(r.Context(), t.ID)
	if err != nil {
		return err
	}
	calls := []domain.Call{}
	for _, id := range t.CallIDs {
		if c, err := s.d.Store.GetCall(r.Context(), id); err == nil {
			calls = append(calls, *c)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ticket": t, "comments": comments, "history": history, "calls": calls,
		"overdue": t.Overdue(time.Now())})
	return nil
}

func (s *Server) patchTicket(w http.ResponseWriter, r *http.Request) error {
	t, err := s.loadTicket(r)
	if err != nil {
		return err
	}
	var p tickets.Patch
	if err := decode(r, &p); err != nil {
		return err
	}
	probe := domain.Ticket{DepartmentID: deref(p.DepartmentID), AssigneeID: deref(p.AssigneeID), ContactID: deref(p.ContactID)}
	if err := s.checkTicketRefs(r, &probe); err != nil {
		return err
	}
	updated, err := s.d.Tickets.Update(r.Context(), t.ID, p, actor(r))
	if err != nil {
		return ticketErr(err)
	}
	writeJSON(w, http.StatusOK, updated)
	return nil
}

func deref(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}
