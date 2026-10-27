package server

import (
	"callcenter-middleware/internal/domain"
	"net/http"
)

func (s *Server) listMembers(w http.ResponseWriter, r *http.Request) error {
	ms, err := s.d.Store.ListMembers(r.Context())
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, ms)
	return nil
}

func (s *Server) putUserQueues(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var ms []domain.QueueMember
	if err := decode(r, &ms); err != nil {
		return err
	}
	if err := s.d.Store.SetUserQueues(r.Context(), id, ms); err != nil {
		return err
	}
	s.audit(r, "set_queues", "user", id, "")
	s.configChanged(r, "member")
	return s.listMembers(w, r)
}

func (s *Server) putQueueMembers(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if _, err := s.d.Store.Queues.Get(r.Context(), id); err != nil {
		return err
	}
	var ms []domain.QueueMember
	if err := decode(r, &ms); err != nil {
		return err
	}
	if err := s.d.Store.SetQueueMembers(r.Context(), id, ms); err != nil {
		return err
	}
	s.audit(r, "set_members", "queue", id, "")
	s.configChanged(r, "member")
	return s.listMembers(w, r)
}

func (s *Server) deleteQueueMembers(r *http.Request, id int64) {
	_ = s.d.Store.DeleteQueueMembers(r.Context(), id)
}
