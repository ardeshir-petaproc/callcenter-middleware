package server

import (
	"net/http"
	"strings"

	"callcenter-middleware/internal/domain"
)

func (s *Server) listRules(w http.ResponseWriter, r *http.Request) error {
	all, err := s.d.Store.Rules.List(r.Context())
	if err != nil {
		return err
	}
	status, out := r.URL.Query().Get("status"), []domain.RoutingRule{}
	for _, rule := range all {
		if status == "" || rule.Status == status {
			out = append(out, rule)
		}
	}
	writeJSON(w, http.StatusOK, out)
	return nil
}

func (s *Server) checkRule(r *http.Request, rule *domain.RoutingRule) error {
	rule.Name = strings.TrimSpace(rule.Name)
	if rule.Name == "" {
		return bad("name is required")
	}
	kinds := map[string]string{domain.TargetUser: "user", domain.TargetJobProfile: "job_profile",
		domain.TargetQueue: "queue", domain.TargetDepartment: "department"}
	kind, ok := kinds[rule.TargetType]
	if !ok || rule.TargetID <= 0 {
		return bad("target_type must be user, job_profile, queue or department with a target_id")
	}
	if err := s.exists(r, kind, rule.TargetID); err != nil {
		return err
	}
	if err := s.exists(r, "job_profile", rule.JobProfileID); err != nil {
		return err
	}
	for _, p := range []*[]string{&rule.Topics, &rule.Keywords, &rule.Languages} {
		if *p == nil {
			*p = []string{}
		}
	}
	return nil
}

// saveRule stores a rule as a draft. Any edit, including to an approved
// rule, needs a fresh approval before it routes calls.
func (s *Server) saveRule(w http.ResponseWriter, r *http.Request) error {
	var rule domain.RoutingRule
	if err := decode(r, &rule); err != nil {
		return err
	}
	if err := s.checkRule(r, &rule); err != nil {
		return err
	}
	rule.TenantID, rule.Status, rule.ApprovedBy, rule.ApprovedAt, rule.UpdatedAt = 0, domain.RuleDraft, 0, "", nowRFC()
	id, _ := pathID(r)
	action := "create"
	if id == 0 {
		rule.ID, rule.Version, rule.CreatedBy = 0, 1, claimsOf(r).UserID
		if rule.Source == "" {
			rule.Source = "manual"
		}
		if err := s.d.Store.Rules.Create(r.Context(), &rule); err != nil {
			return err
		}
	} else {
		prev, err := s.d.Store.Rules.Get(r.Context(), id)
		if err != nil {
			return err
		}
		rule.ID, rule.Version, rule.CreatedBy, rule.Source = id, prev.Version+1, prev.CreatedBy, prev.Source
		if err := s.d.Store.Rules.Update(r.Context(), &rule); err != nil {
			return err
		}
		action = "update"
	}
	s.audit(r, "routing_rule."+action, "routing_rule", rule.ID, rule.Name)
	s.revision(r, s.d.Store.Rules, action, rule.ID)
	writeJSON(w, http.StatusOK, rule)
	return nil
}

func (s *Server) deleteRule(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if err := s.d.Store.Rules.Delete(r.Context(), id); err != nil {
		return err
	}
	s.audit(r, "routing_rule.delete", "routing_rule", id, "")
	s.revision(r, s.d.Store.Rules, "delete", id)
	ok(w)
	return nil
}

// reviewRule approves, rejects or retires a rule (routing.approve).
func (s *Server) reviewRule(status string) handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		var req struct {
			Note string `json:"note"`
		}
		_ = decode(r, &req)
		rule, err := s.d.Store.Rules.Get(r.Context(), id)
		if err != nil {
			return err
		}
		if status == domain.RuleApproved && rule.Status != domain.RuleDraft {
			return bad("only draft rules can be approved")
		}
		rule.Status, rule.UpdatedAt = status, nowRFC()
		if status == domain.RuleApproved {
			rule.ApprovedBy, rule.ApprovedAt = claimsOf(r).UserID, rule.UpdatedAt
		}
		if err := s.d.Store.Rules.Update(r.Context(), rule); err != nil {
			return err
		}
		s.audit(r, "routing_rule."+status, "routing_rule", id, req.Note)
		s.revision(r, s.d.Store.Rules, status, id)
		writeJSON(w, http.StatusOK, rule)
		return nil
	}
}
