package server

import (
	"net/http"
	"strings"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/routing"

	"github.com/go-chi/chi/v5"
)

func (s *Server) routingRoutes(r chi.Router) {
	manage, approve := requirePerm(domain.PermRoutingManage), requirePerm(domain.PermRoutingApprove)
	mountCRUD(r, "/routing/topics", s, s.d.Store.Topics, crudOpts[domain.Topic]{entity: "topic",
		writePerm: domain.PermRoutingManage, history: true, prepare: prepareTopic,
		check: func(r *http.Request, t *domain.Topic) error { return s.exists(r, "department", t.DepartmentID) }})
	r.With(anyPerm(domain.PermRoutingManage, domain.PermRoutingApprove)).Get("/routing/rules", handler(s.listRules).ServeHTTP)
	r.With(manage).Post("/routing/rules", handler(s.saveRule).ServeHTTP)
	r.With(manage).Put("/routing/rules/{id}", handler(s.saveRule).ServeHTTP)
	r.With(manage).Delete("/routing/rules/{id}", handler(s.deleteRule).ServeHTTP)
	r.With(approve).Post("/routing/rules/{id}/approve", handler(s.reviewRule(domain.RuleApproved)).ServeHTTP)
	r.With(approve).Post("/routing/rules/{id}/reject", handler(s.reviewRule(domain.RuleRejected)).ServeHTTP)
	r.With(approve).Post("/routing/rules/{id}/retire", handler(s.reviewRule(domain.RuleRetired)).ServeHTTP)
	r.With(manage).Post("/routing/generate", handler(s.generateRules).ServeHTTP)
	r.With(manage).Post("/routing/simulate", handler(s.simulateRoute).ServeHTTP)
	r.With(anyPerm(domain.PermRoutingManage, domain.PermReportsView)).Get("/routing/decisions", handler(s.listDecisions).ServeHTTP)
}

// anyPerm admits callers holding at least one of the permissions.
func anyPerm(perms ...domain.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, p := range perms {
				if can(r, p) {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeErr(w, errForbidden)
		})
	}
}

func prepareTopic(t *domain.Topic) error {
	t.Name = strings.TrimSpace(t.Name)
	if t.Name == "" {
		return bad("name is required")
	}
	if t.Keywords == nil {
		t.Keywords = []string{}
	}
	return nil
}

// simulateRoute shows where a request would go right now and why, without
// touching any call.
func (s *Server) simulateRoute(w http.ResponseWriter, r *http.Request) error {
	var req domain.RouteRequest
	if err := decode(r, &req); err != nil {
		return err
	}
	if strings.TrimSpace(req.Text+req.Topic) == "" {
		return bad("text or topic is required")
	}
	req.CallID = ""
	d, err := s.d.Router.Decide(r.Context(), req)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, d)
	return nil
}

func (s *Server) listDecisions(w http.ResponseWriter, r *http.Request) error {
	out, err := s.d.Store.Decisions(r.Context(), r.URL.Query().Get("call_id"), qInt64(r, "rule_id"), qInt(r, "limit", 100))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

// generateRules drafts rules from a job profile (AI when configured).
// Drafts never route calls until approved.
func (s *Server) generateRules(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		JobProfileID int64 `json:"job_profile_id"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	job, err := s.d.Store.JobProfiles.Get(r.Context(), req.JobProfileID)
	if err != nil {
		return err
	}
	topics, err := s.d.Store.Topics.List(r.Context())
	if err != nil {
		return err
	}
	var drafter routing.Drafter
	if s.d.OpenAI != nil && s.d.Config.OpenAIAPIKey != "" {
		drafter = s.d.OpenAI
	}
	rules, method := routing.Generate(r.Context(), drafter, job, topics)
	uid := claimsOf(r).UserID
	for i := range rules {
		rules[i].CreatedBy, rules[i].Version, rules[i].UpdatedAt = uid, 1, nowRFC()
		if err := s.d.Store.Rules.Create(r.Context(), &rules[i]); err != nil {
			return err
		}
		s.revision(r, s.d.Store.Rules, "create", rules[i].ID)
	}
	s.audit(r, "routing.generate", "job_profile", job.ID, method)
	s.notifyApprovers(r, len(rules), "rules drafted from job profile "+job.Title)
	writeJSON(w, http.StatusOK, map[string]any{"method": method, "rules": rules})
	return nil
}
