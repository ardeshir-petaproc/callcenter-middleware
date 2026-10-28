package server

import "net/http"

// acknowledgeEvaluation lets the evaluated agent confirm they have read
// their evaluation, optionally with a comment.
func (s *Server) acknowledgeEvaluation(w http.ResponseWriter, r *http.Request) error {
	e, err := s.loadEvaluation(r)
	if err != nil {
		return err
	}
	if e.AgentID != claimsOf(r).UserID || e.Status != "final" {
		return errForbidden
	}
	var req struct {
		Comment string `json:"comment"`
	}
	_ = decode(r, &req)
	e.Acknowledged, e.AgentComment, e.UpdatedAt = true, req.Comment, nowRFC()
	if err := s.d.Store.Evaluations.Update(r.Context(), e); err != nil {
		return err
	}
	s.audit(r, "qa.acknowledge", "qa_evaluation", e.ID, "")
	writeJSON(w, http.StatusOK, e)
	return nil
}

// qaSummary aggregates final evaluations per agent.
func (s *Server) qaSummary(w http.ResponseWriter, r *http.Request) error {
	all, err := s.d.Store.Evaluations.List(r.Context())
	if err != nil {
		return err
	}
	type row struct {
		AgentID     int64   `json:"agent_id"`
		Evaluations int     `json:"evaluations"`
		AvgScore    float64 `json:"avg_score"`
		Passed      int     `json:"passed"`
		AutoFailed  int     `json:"auto_failed"`
	}
	rows := map[int64]*row{}
	var order []int64
	for _, e := range all {
		if e.Status != "final" {
			continue
		}
		rw := rows[e.AgentID]
		if rw == nil {
			rw = &row{AgentID: e.AgentID}
			rows[e.AgentID], order = rw, append(order, e.AgentID)
		}
		rw.AvgScore = (rw.AvgScore*float64(rw.Evaluations) + e.Score) / float64(rw.Evaluations+1)
		rw.Evaluations++
		if e.Passed {
			rw.Passed++
		}
		if e.AutoFailed {
			rw.AutoFailed++
		}
	}
	out := []row{}
	for _, id := range order {
		rw := *rows[id]
		rw.AvgScore = float64(int(rw.AvgScore*10+0.5)) / 10
		out = append(out, rw)
	}
	writeJSON(w, http.StatusOK, map[string]any{"agents": out, "scale": "percent"})
	return nil
}
