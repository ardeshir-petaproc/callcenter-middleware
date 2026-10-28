package server

import (
	"errors"
	"net/http"
	"strconv"

	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"

	"github.com/go-chi/chi/v5"
)

func (s *Server) consultRoutes(r chi.Router) {
	r.Post("/calls/{id}/consult", handler(s.callConsult).ServeHTTP)
	r.Post("/calls/{id}/consult/complete", handler(s.consultStep("transfer_complete", s.d.Core.CompleteTransfer)).ServeHTTP)
	r.Post("/calls/{id}/consult/conference", handler(s.consultStep("conference", s.d.Core.Conference)).ServeHTTP)
	r.Post("/calls/{id}/consult/cancel", handler(s.consultStep("consult_cancel", s.d.Core.CancelConsult)).ServeHTTP)
}

// callConsult starts a warm transfer to an agent ({"type":"agent","target":"<user id>"})
// or an outside number ({"type":"external","target":"+1555..."}).
func (s *Server) callConsult(w http.ResponseWriter, r *http.Request) error {
	id, err := s.liveCallFor(r)
	if err != nil {
		return err
	}
	var a domain.Action
	if err := decode(r, &a); err != nil {
		return err
	}
	if a.Type == domain.ActionAgent {
		if err := s.exists(r, "user", targetID(a.Target)); err != nil {
			return err
		}
	}
	if err := s.d.Core.Consult(id, a); err != nil {
		return consultErr(err)
	}
	s.audit(r, "consult", "call", id, string(a.Type)+":"+a.Target)
	ok(w)
	return nil
}

func (s *Server) consultStep(action string, fn func(string) error) handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := s.liveCallFor(r)
		if err != nil {
			return err
		}
		if err := fn(id); err != nil {
			return consultErr(err)
		}
		s.audit(r, action, "call", id, "")
		ok(w)
		return nil
	}
}

func consultErr(err error) error {
	if errors.Is(err, core.ErrInvalidState) || errors.Is(err, core.ErrCallNotFound) {
		return err
	}
	return bad(err.Error())
}

func targetID(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	if v <= 0 {
		return -1
	}
	return v
}
