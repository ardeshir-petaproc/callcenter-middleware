package server

import (
	"callcenter-middleware/internal/domain"
	"encoding/json"
	"net/http"
)

func (s *Server) saveNewUser(w http.ResponseWriter, r *http.Request, req *userReq) error {
	if err := s.prepareUser(r, req, true); err != nil {
		return err
	}
	u := req.User
	if err := s.d.Store.CreateUser(r.Context(), &u); err != nil {
		return err
	}
	if req.Queues != nil {
		if err := s.d.Store.SetUserQueues(r.Context(), u.ID, req.Queues); err != nil {
			return err
		}
	}
	s.audit(r, "create", "user", u.ID, u.Username)
	s.userRevision(r, &u, "create")
	s.configChanged(r, "user")
	writeJSON(w, http.StatusCreated, u)
	return nil
}

// userRevision keeps the history of employee profile changes.
func (s *Server) userRevision(r *http.Request, u *domain.User, action string) {
	raw, _ := json.Marshal(u)
	p := principalOf(r)
	_ = s.d.Store.AddRevision(r.Context(), domain.Revision{Entity: "user", EntityID: u.ID, Action: action,
		Data: string(raw), ChangedBy: p.user.ID, ChangedByName: p.user.DisplayName})
}
