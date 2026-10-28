package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"
)

func (s *Server) commentTicket(w http.ResponseWriter, r *http.Request) error {
	t, err := s.loadTicket(r)
	if err != nil {
		return err
	}
	var req struct {
		Body     string `json:"body"`
		Internal bool   `json:"internal"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	c, err := s.d.Tickets.Comment(r.Context(), t.ID, req.Body, req.Internal, actor(r))
	if err != nil {
		return ticketErr(err)
	}
	writeJSON(w, http.StatusCreated, c)
	return nil
}

func (s *Server) linkTicketCall(w http.ResponseWriter, r *http.Request) error {
	t, err := s.loadTicket(r)
	if err != nil {
		return err
	}
	var req struct {
		CallID string `json:"call_id"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	updated, err := s.d.Tickets.LinkCall(r.Context(), t.ID, req.CallID, actor(r))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, updated)
	return nil
}

// ticketFromCall opens a ticket prefilled from a call: caller, contact, AI
// summary and transcript excerpt.
func (s *Server) ticketFromCall(w http.ResponseWriter, r *http.Request) error {
	call, err := s.loadCall(r)
	if err != nil {
		return err
	}
	var req domain.Ticket
	if err := decode(r, &req); err != nil {
		return err
	}
	t := &domain.Ticket{Subject: req.Subject, Category: req.Category, Priority: req.Priority, Source: "call",
		DepartmentID: req.DepartmentID, AssigneeID: req.AssigneeID, ContactID: call.ContactID,
		RequesterName: call.CallerName, RequesterPhone: call.CallerNumber, CallIDs: []string{call.ID}}
	if t.Subject == "" {
		t.Subject = fmt.Sprintf("Call from %s", firstNonEmpty(call.CallerName, call.CallerNumber))
	}
	t.Description = firstNonEmpty(req.Description, s.callDigest(r, call))
	if t.AssigneeID == 0 {
		t.AssigneeID = claimsOf(r).UserID
	}
	if err := s.checkTicketRefs(r, t); err != nil {
		return err
	}
	if err := s.d.Tickets.Create(r.Context(), t, actor(r)); err != nil {
		return ticketErr(err)
	}
	writeJSON(w, http.StatusCreated, t)
	return nil
}

// callDigest summarizes a call for a ticket description.
func (s *Server) callDigest(r *http.Request, call *domain.Call) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Call %s at %s (%s, %s).\n", call.ID[:8], call.StartedAt.Format(time.RFC822),
		call.Direction, call.Outcome)
	if call.Summary != "" {
		fmt.Fprintf(&b, "\nSummary: %s\n", call.Summary)
	}
	if lines, err := s.d.Store.Transcript(r.Context(), call.ID); err == nil && len(lines) > 0 {
		b.WriteString("\nTranscript excerpt:\n")
		for i, l := range lines {
			if i == 12 {
				b.WriteString("…\n")
				break
			}
			fmt.Fprintf(&b, "%s: %s\n", l.Speaker, l.Text)
		}
	}
	return b.String()
}

func (s *Server) ticketStats(w http.ResponseWriter, r *http.Request) error {
	f := reportFilter(r)
	st, err := s.d.Store.TicketStatistics(r.Context(), f.From, f.To)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, st)
	return nil
}

func prepareSLA(p *domain.SLAPolicy) error {
	switch p.Priority {
	case domain.PriorityLow, domain.PriorityNormal, domain.PriorityHigh, domain.PriorityUrgent:
	default:
		return bad("priority must be low, normal, high or urgent")
	}
	if p.FirstResponseMinutes < 0 || p.ResolutionMinutes < 0 {
		return bad("targets must be positive")
	}
	return nil
}
