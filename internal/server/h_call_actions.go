package server

import (
	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// liveCallFor returns the live call if the requester may control it.
func (s *Server) liveCallFor(r *http.Request) (string, error) {
	id := chi.URLParam(r, "id")
	v, ok := s.d.Core.LiveCall(id)
	if !ok {
		return "", core.ErrCallNotFound
	}
	if !isSup(r) && v.AgentID != claimsOf(r).UserID {
		return "", errForbidden
	}
	return id, nil
}

func (s *Server) callHangup(w http.ResponseWriter, r *http.Request) error {
	id, err := s.liveCallFor(r)
	if err != nil {
		return err
	}
	if err := s.d.Core.Hangup(id); err != nil {
		return err
	}
	s.audit(r, "hangup", "call", id, "")
	ok(w)
	return nil
}

func (s *Server) callTransfer(w http.ResponseWriter, r *http.Request) error {
	id, err := s.liveCallFor(r)
	if err != nil {
		return err
	}
	var a domain.Action
	if err := decode(r, &a); err != nil {
		return err
	}
	if err := validAction(a); err != nil {
		return err
	}
	if err := s.d.Core.Transfer(id, a); err != nil {
		if err.Error() == "unsupported transfer target" {
			return bad(err.Error())
		}
		return err
	}
	s.audit(r, "transfer", "call", id, string(a.Type)+":"+a.Target)
	ok(w)
	return nil
}

func (s *Server) callHold(w http.ResponseWriter, r *http.Request) error {
	id, err := s.liveCallFor(r)
	if err != nil {
		return err
	}
	var req struct {
		On bool `json:"on"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if err := s.d.Core.Hold(id, req.On); err != nil {
		return err
	}
	ok(w)
	return nil
}

func (s *Server) callRecord(w http.ResponseWriter, r *http.Request) error {
	id, err := s.liveCallFor(r)
	if err != nil {
		return err
	}
	if err := s.d.Core.Record(id); err != nil {
		return err
	}
	s.audit(r, "record", "call", id, "")
	ok(w)
	return nil
}

func (s *Server) callMonitor(w http.ResponseWriter, r *http.Request) error {
	id := chi.URLParam(r, "id")
	var req struct {
		Mode string `json:"mode"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	if err := s.d.Core.Monitor(id, claimsOf(r).UserID, req.Mode); err != nil {
		if err == core.ErrCallNotFound || err == core.ErrInvalidState {
			return err
		}
		return bad(err.Error())
	}
	s.audit(r, "monitor_"+req.Mode, "call", id, "")
	ok(w)
	return nil
}
