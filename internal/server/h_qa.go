package server

import (
	"net/http"
	"sort"
	"strings"

	"callcenter-middleware/internal/domain"

	"github.com/go-chi/chi/v5"
)

func (s *Server) qaRoutes(r chi.Router) {
	qa := requirePerm(domain.PermQAEvaluate)
	mountCRUD(r, "/qa/scorecards", s, s.d.Store.Scorecards, crudOpts[domain.QAScorecard]{entity: "qa_scorecard",
		writePerm: domain.PermQAEvaluate, history: true, prepare: prepareScorecard})
	r.Get("/qa/evaluations", handler(s.listEvaluations).ServeHTTP)
	r.Get("/qa/evaluations/{id}", handler(s.getEvaluation).ServeHTTP)
	r.With(qa).Post("/qa/evaluations", handler(s.saveEvaluation).ServeHTTP)
	r.With(qa).Put("/qa/evaluations/{id}", handler(s.saveEvaluation).ServeHTTP)
	r.Post("/qa/evaluations/{id}/acknowledge", handler(s.acknowledgeEvaluation).ServeHTTP)
	r.With(anyPerm(domain.PermQAEvaluate, domain.PermReportsView)).Get("/qa/summary", handler(s.qaSummary).ServeHTTP)
}

func prepareScorecard(sc *domain.QAScorecard) error {
	if strings.TrimSpace(sc.Name) == "" || len(sc.Criteria) == 0 {
		return bad("name and at least one criterion are required")
	}
	seen := map[string]bool{}
	for i := range sc.Criteria {
		c := &sc.Criteria[i]
		if c.Key == "" {
			c.Key = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(c.Label), " ", "_"))
		}
		if c.Key == "" || seen[c.Key] {
			return bad("criteria need unique keys or labels")
		}
		seen[c.Key] = true
		c.Weight, c.MaxScore = max(c.Weight, 1), max(c.MaxScore, 1)
	}
	if sc.PassScore < 0 || sc.PassScore > 100 {
		return bad("pass_score must be between 0 and 100")
	}
	return nil
}

// evaluationVisible: reviewers see everything; agents see their own final
// evaluations.
func evaluationVisible(r *http.Request, e *domain.QAEvaluation) bool {
	return can(r, domain.PermQAEvaluate) || (e.AgentID == claimsOf(r).UserID && e.Status == "final")
}

func (s *Server) listEvaluations(w http.ResponseWriter, r *http.Request) error {
	all, err := s.d.Store.Evaluations.List(r.Context())
	if err != nil {
		return err
	}
	agent, callID := qInt64(r, "agent_id"), r.URL.Query().Get("call_id")
	out := []domain.QAEvaluation{}
	for i := range all {
		e := &all[i]
		if !evaluationVisible(r, e) || (agent > 0 && e.AgentID != agent) || (callID != "" && e.CallID != callID) {
			continue
		}
		out = append(out, *e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	writeJSON(w, http.StatusOK, out)
	return nil
}

func (s *Server) loadEvaluation(r *http.Request) (*domain.QAEvaluation, error) {
	id, err := pathID(r)
	if err != nil {
		return nil, err
	}
	e, err := s.d.Store.Evaluations.Get(r.Context(), id)
	if err != nil {
		return nil, err
	}
	if !evaluationVisible(r, e) {
		return nil, errForbidden
	}
	return e, nil
}

func (s *Server) getEvaluation(w http.ResponseWriter, r *http.Request) error {
	e, err := s.loadEvaluation(r)
	if err != nil {
		return err
	}
	sc, _ := s.d.Store.Scorecards.Get(r.Context(), e.ScorecardID)
	writeJSON(w, http.StatusOK, map[string]any{"evaluation": e, "scorecard": sc})
	return nil
}

// saveEvaluation grades a call against a scorecard. Final evaluations
// become visible to the agent.
func (s *Server) saveEvaluation(w http.ResponseWriter, r *http.Request) error {
	var e domain.QAEvaluation
	if err := decode(r, &e); err != nil {
		return err
	}
	call, err := s.d.Store.GetCall(r.Context(), e.CallID)
	if err != nil {
		return bad("call does not exist in this organization")
	}
	sc, err := s.d.Store.Scorecards.Get(r.Context(), e.ScorecardID)
	if err != nil {
		return bad("scorecard does not exist in this organization")
	}
	if e.Status != "final" {
		e.Status = "draft"
	}
	if e.Scores == nil {
		e.Scores = map[string]int{}
	}
	e.TenantID, e.AgentID, e.EvaluatorID, e.UpdatedAt = 0, call.AgentID, claimsOf(r).UserID, nowRFC()
	e.Grade(sc)
	id, _ := pathID(r)
	if id == 0 {
		e.ID, e.CreatedAt, e.Acknowledged, e.AgentComment = 0, e.UpdatedAt, false, ""
		err = s.d.Store.Evaluations.Create(r.Context(), &e)
	} else {
		prev, gerr := s.d.Store.Evaluations.Get(r.Context(), id)
		if gerr != nil {
			return gerr
		}
		e.ID, e.CreatedAt, e.Acknowledged, e.AgentComment = id, prev.CreatedAt, prev.Acknowledged, prev.AgentComment
		err = s.d.Store.Evaluations.Update(r.Context(), &e)
	}
	if err != nil {
		return err
	}
	s.audit(r, "qa.evaluate", "call", e.CallID, sc.Name)
	writeJSON(w, http.StatusOK, e)
	return nil
}
