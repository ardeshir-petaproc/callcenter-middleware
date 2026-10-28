package server

import (
	"callcenter-middleware/internal/domain"
	"net/http"
	"time"
)

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
	if _, err := s.d.Store.GetUser(r.Context(), id); err != nil {
		return err // other tenants' users are simply not found
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
