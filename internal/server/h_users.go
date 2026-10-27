package server

import (
	"net/http"
	"strings"

	"callcenter-middleware/internal/auth"
	"callcenter-middleware/internal/domain"
)

type userReq struct {
	domain.User
	Password string               `json:"password"`
	Queues   []domain.QueueMember `json:"queues"`
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) error {
	users, err := s.d.Store.ListUsers(r.Context())
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, users)
	return nil
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	u, err := s.d.Store.GetUser(r.Context(), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, u)
	return nil
}

func (s *Server) prepareUser(req *userReq, creating bool) error {
	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" {
		return bad("username is required")
	}
	if req.Role == "" {
		req.Role = domain.RoleAgent
	}
	if !req.Role.Valid() {
		return bad("role must be admin, supervisor or agent")
	}
	if creating && len(req.Password) < 6 {
		return bad("password must be at least 6 characters")
	}
	if req.Password != "" {
		if len(req.Password) < 6 {
			return bad("password must be at least 6 characters")
		}
		h, err := auth.HashPassword(req.Password)
		if err != nil {
			return err
		}
		req.PasswordHash = h
	}
	if req.DisplayName == "" {
		req.DisplayName = req.Username
	}
	if req.Skills == nil {
		req.Skills = []string{}
	}
	return nil
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) error {
	var req userReq
	req.Enabled = true
	if err := decode(r, &req); err != nil {
		return err
	}
	if err := s.prepareUser(&req, true); err != nil {
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
	s.configChanged(r, "user")
	writeJSON(w, http.StatusCreated, u)
	return nil
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var req userReq
	if err := decode(r, &req); err != nil {
		return err
	}
	req.ID = id
	if err := s.prepareUser(&req, false); err != nil {
		return err
	}
	if id == claimsOf(r).UserID && (req.Role != domain.RoleAdmin || !req.Enabled) {
		return bad("you cannot demote or disable yourself")
	}
	u := req.User
	if err := s.d.Store.UpdateUser(r.Context(), &u); err != nil {
		return err
	}
	if req.Queues != nil {
		if err := s.d.Store.SetUserQueues(r.Context(), id, req.Queues); err != nil {
			return err
		}
	}
	s.audit(r, "update", "user", id, u.Username)
	s.configChanged(r, "user")
	fresh, err := s.d.Store.GetUser(r.Context(), id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, fresh)
	return nil
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if id == claimsOf(r).UserID {
		return bad("you cannot delete yourself")
	}
	if err := s.d.Store.DeleteUser(r.Context(), id); err != nil {
		return err
	}
	_ = s.d.Store.SetUserQueues(r.Context(), id, nil)
	s.audit(r, "delete", "user", id, "")
	s.configChanged(r, "user")
	ok(w)
	return nil
}

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
