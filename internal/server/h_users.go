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

func (s *Server) prepareUser(r *http.Request, req *userReq, creating bool) error {
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
	// Joow-linked employees sign in with Joow and need no local password.
	if creating && req.JoowAccountID == "" && len(req.Password) < 6 {
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
	switch req.Status {
	case "":
		req.Status = domain.UserActive
	case domain.UserActive, domain.UserInvited, domain.UserDisabled:
	default:
		return bad("invalid status")
	}
	req.PlatformAdmin = false // only set at bootstrap / by platform tooling
	for kind, id := range map[string]int64{"user": req.ManagerID, "job_profile": req.JobProfileID,
		"custom_role": req.CustomRoleID, "schedule": req.ScheduleID} {
		if err := s.exists(r, kind, id); err != nil {
			return err
		}
	}
	for _, d := range req.DepartmentIDs {
		if err := s.exists(r, "department", d); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) error {
	var req userReq
	req.Enabled = true
	if err := decode(r, &req); err != nil {
		return err
	}
	return s.saveNewUser(w, r, &req)
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
	if err := s.prepareUser(r, &req, false); err != nil {
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
	s.userRevision(r, &u, "update")
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
