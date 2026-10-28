package integration

import (
	"testing"

	"callcenter-middleware/internal/domain"
)

func TestRouting_ApprovalWorkflowAndSimulation(t *testing.T) {
	s := newStack(t, 40)
	admin, sara := s.login("admin", "admin123"), s.login("sara", "agent123")
	var jobs []domain.JobProfile
	s.call("GET", "/job-profiles", admin, nil, &jobs)
	sales := jobs[0]

	// The pricing rule is a draft: simulation does not use it.
	var d domain.RoutingDecision
	s.call("POST", "/routing/simulate", sara, map[string]any{"text": "how much does the pro plan cost", "topic": "Sales"}, &d)
	if d.RuleID != 0 {
		t.Fatalf("draft used: %+v", d)
	}
	var drafts []domain.RoutingRule
	s.call("GET", "/routing/rules?status=draft", sara, nil, &drafts)
	if len(drafts) != 1 {
		t.Fatalf("drafts: %+v", drafts)
	}
	// Supervisors draft rules; only routing.approve holders activate them.
	if code := s.call("POST", "/routing/rules/"+itoa(drafts[0].ID)+"/approve", sara, nil, nil); code != 403 {
		t.Fatalf("supervisor approve: %d", code)
	}
	var approved domain.RoutingRule
	if code := s.call("POST", "/routing/rules/"+itoa(drafts[0].ID)+"/approve", admin, nil, &approved); code != 200 ||
		approved.Status != "approved" || approved.ApprovedBy == 0 {
		t.Fatalf("approve: %d %+v", code, approved)
	}
	s.call("POST", "/me/status", s.login("alice", "agent123"), map[string]any{"status": "available"}, nil)
	s.call("POST", "/routing/simulate", sara, map[string]any{"text": "how much does the pro plan cost", "topic": "Sales"}, &d)
	if d.RuleID != approved.ID || d.TargetName != "Alice Moreno" || len(d.Steps) == 0 {
		t.Fatalf("approved rule should route to alice: %+v", d)
	}
	// Editing an approved rule sends it back for approval.
	approved.Priority = 3
	var edited domain.RoutingRule
	s.call("PUT", "/routing/rules/"+itoa(approved.ID), sara, approved, &edited)
	if edited.Status != "draft" || edited.Version != 2 || edited.ApprovedBy != 0 {
		t.Fatalf("edit must require re-approval: %+v", edited)
	}
	// Rules generated from a job description are drafts with a rationale.
	var gen struct {
		Method string               `json:"method"`
		Rules  []domain.RoutingRule `json:"rules"`
	}
	s.call("POST", "/routing/generate", sara, map[string]any{"job_profile_id": sales.ID}, &gen)
	if gen.Method != "template" || len(gen.Rules) == 0 || gen.Rules[0].Status != "draft" || gen.Rules[0].Rationale == "" {
		t.Fatalf("generate: %+v", gen)
	}
	var revs []domain.Revision
	s.call("GET", "/revisions?entity=routing_rule&entity_id="+itoa(approved.ID), admin, nil, &revs)
	if len(revs) < 2 {
		t.Fatalf("rule history: %+v", revs)
	}
	// Agents cannot see or change routing; other tenants see nothing.
	if code := s.call("GET", "/routing/rules", s.login("bob", "agent123"), nil, nil); code != 403 {
		t.Fatalf("agent read: %d", code)
	}
	var other []domain.RoutingRule
	s.call("GET", "/routing/rules", s.loginTenant("admin", "admin123", "parsian"), nil, &other)
	if len(other) != 0 {
		t.Fatalf("tenant leak: %+v", other)
	}
}
