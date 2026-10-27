package server

import (
	"net/http"
	"strings"
	"time"

	"callcenter-middleware/internal/auth"
	"callcenter-middleware/internal/domain"
)

func (s *Server) login(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	u, err := s.d.Store.GetUserByUsername(r.Context(), strings.TrimSpace(req.Username))
	if err != nil || !u.Enabled || !auth.CheckPassword(u.PasswordHash, req.Password) {
		// Constant-ish failure path to avoid username probing.
		time.Sleep(200 * time.Millisecond)
		return errStatus(http.StatusUnauthorized, "invalid username or password")
	}
	tok, exp, err := s.d.Issuer.Issue(u)
	if err != nil {
		return err
	}
	_ = s.d.Store.Audit(r.Context(), domain.AuditEntry{UserID: u.ID, Username: u.Username, Action: "login", Entity: "user", EntityID: itoa(u.ID)})
	writeJSON(w, http.StatusOK, map[string]any{"token": tok, "expires_at": exp.UTC(), "user": u})
	return nil
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) error {
	u, err := s.d.Store.GetUser(r.Context(), claimsOf(r).UserID)
	if err != nil {
		return errStatus(http.StatusUnauthorized, "user no longer exists")
	}
	resp := map[string]any{"user": u, "agent": nil}
	if st, ok := s.d.Core.Agent(u.ID); ok {
		resp["agent"] = st
	}
	writeJSON(w, http.StatusOK, resp)
	return nil
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Current string `json:"current_password"`
		New     string `json:"new_password"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	u, err := s.d.Store.GetUser(r.Context(), claimsOf(r).UserID)
	if err != nil {
		return err
	}
	if !auth.CheckPassword(u.PasswordHash, req.Current) {
		return errStatus(http.StatusBadRequest, "current password is incorrect")
	}
	if len(req.New) < 8 {
		return bad("new password must be at least 8 characters")
	}
	if u.PasswordHash, err = auth.HashPassword(req.New); err != nil {
		return err
	}
	if err := s.d.Store.UpdateUser(r.Context(), u); err != nil {
		return err
	}
	s.audit(r, "change_password", "user", u.ID, "")
	ok(w)
	return nil
}

type statusReq struct {
	Status domain.AgentStatus `json:"status"`
	Reason string             `json:"reason"`
}

func (s *Server) applyStatus(w http.ResponseWriter, r *http.Request, userID int64) error {
	var req statusReq
	if err := decode(r, &req); err != nil {
		return err
	}
	if err := s.d.Core.SetAgentStatus(userID, req.Status, req.Reason); err != nil {
		if err.Error() == "status must be available, paused or offline" {
			return bad(err.Error())
		}
		return err
	}
	st, _ := s.d.Core.Agent(userID)
	writeJSON(w, http.StatusOK, st)
	return nil
}

func (s *Server) setMyStatus(w http.ResponseWriter, r *http.Request) error {
	return s.applyStatus(w, r, claimsOf(r).UserID)
}

func (s *Server) forceStatus(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	s.audit(r, "force_status", "user", id, "")
	return s.applyStatus(w, r, id)
}

func (s *Server) myStats(w http.ResponseWriter, r *http.Request) error {
	now := time.Now()
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	sum, err := s.d.Store.ReportSummary(r.Context(), domain.CallFilter{From: start, AgentID: claimsOf(r).UserID})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, sum)
	return nil
}

func (s *Server) myDial(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		Number string `json:"number"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	id, err := s.d.Core.Dial(claimsOf(r).UserID, req.Number, 0)
	if err != nil {
		return dialErr(err)
	}
	s.audit(r, "dial", "call", id, req.Number)
	writeJSON(w, http.StatusOK, map[string]string{"call_id": id})
	return nil
}

func dialErr(err error) error {
	switch err.Error() {
	case "invalid number", "agent has no extension":
		return bad(err.Error())
	}
	return err
}
