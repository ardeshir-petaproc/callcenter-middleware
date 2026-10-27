package server

import (
	"net/http"

	"callcenter-middleware/internal/domain"

	"github.com/go-chi/chi/v5"
)

func callFilter(r *http.Request) domain.CallFilter {
	q := r.URL.Query()
	return domain.CallFilter{
		From: qTime(r, "from", false), To: qTime(r, "to", true),
		QueueID: qInt64(r, "queue_id"), AgentID: qInt64(r, "agent_id"),
		Outcome: q.Get("outcome"), Direction: q.Get("direction"), Search: q.Get("q"),
		Limit: qInt(r, "limit", 50), Offset: qInt(r, "offset", 0),
	}
}

func (s *Server) listCalls(w http.ResponseWriter, r *http.Request) error {
	f := callFilter(r)
	if !isSup(r) {
		f.AgentID = claimsOf(r).UserID
	}
	items, total, err := s.d.Store.ListCalls(r.Context(), f)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items, "total": total})
	return nil
}

// loadCall fetches a call record and enforces agent ownership.
func (s *Server) loadCall(r *http.Request) (*domain.Call, error) {
	call, err := s.d.Store.GetCall(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		return nil, err
	}
	if !isSup(r) && call.AgentID != claimsOf(r).UserID {
		return nil, errForbidden
	}
	return call, nil
}

func (s *Server) getCall(w http.ResponseWriter, r *http.Request) error {
	call, err := s.loadCall(r)
	if err != nil {
		return err
	}
	ctx := r.Context()
	resp := map[string]any{"call": call, "live": nil, "contact": nil}
	if v, ok := s.d.Core.LiveCall(call.ID); ok {
		resp["live"] = v
	}
	if resp["transcript"], err = s.d.Store.Transcript(ctx, call.ID); err != nil {
		return err
	}
	if resp["events"], err = s.d.Store.CallEvents(ctx, call.ID); err != nil {
		return err
	}
	if call.ContactID > 0 {
		if c, err := s.d.Store.Contacts.Get(ctx, call.ContactID); err == nil {
			resp["contact"] = c
		}
	}
	writeJSON(w, http.StatusOK, resp)
	return nil
}

func (s *Server) patchCall(w http.ResponseWriter, r *http.Request) error {
	call, err := s.loadCall(r)
	if err != nil {
		return err
	}
	var req struct {
		Disposition *string   `json:"disposition"`
		Notes       *string   `json:"notes"`
		Tags        *[]string `json:"tags"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if req.Disposition != nil {
		call.Disposition = *req.Disposition
	}
	if req.Notes != nil {
		call.Notes = *req.Notes
	}
	if req.Tags != nil {
		call.Tags = *req.Tags
	}
	if err := s.d.Store.UpdateCallWrapup(r.Context(), call.ID, call.Disposition, call.Notes, call.Tags); err != nil {
		return err
	}
	s.d.Hub.Publish(domain.Event{Type: domain.EvCallEnded, Data: call})
	writeJSON(w, http.StatusOK, call)
	return nil
}
