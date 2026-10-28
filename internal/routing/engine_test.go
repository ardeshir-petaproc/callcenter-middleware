package routing_test

import (
	"context"
	"strconv"
	"strings"
	"testing"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/routing"
	"callcenter-middleware/internal/seed"
	"callcenter-middleware/internal/store"
)

type live map[int64]domain.AgentStatus

func (l live) Agent(id int64) (domain.AgentState, bool) {
	st, ok := l[id]
	return domain.AgentState{UserID: id, TenantID: 1, Status: st}, ok
}

func setup(t *testing.T) (context.Context, *routing.Engine, map[string]int64, live) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	ctx := store.ForTenant(context.Background(), 1)
	if err := seed.EnsureAdmin(ctx, st, "admin", "admin123"); err != nil {
		t.Fatal(err)
	}
	if err := seed.Demo(ctx, st); err != nil {
		t.Fatal(err)
	}
	users, _ := st.ListUsers(ctx)
	ids := map[string]int64{}
	for _, u := range users {
		ids[u.Username] = u.ID
	}
	l := live{}
	return ctx, &routing.Engine{Store: st, Live: l}, ids, l
}

func decide(t *testing.T, ctx context.Context, e *routing.Engine, req domain.RouteRequest) *domain.RoutingDecision {
	t.Helper()
	d, err := e.Decide(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestDecide_RulesAvailabilityAndEscalation(t *testing.T) {
	ctx, e, ids, l := setup(t)
	l[ids["carol"]], l[ids["dave"]], l[ids["sara"]] = domain.AgentAvailable, domain.AgentBusy, domain.AgentAvailable

	d := decide(t, ctx, e, domain.RouteRequest{Text: "سلام، می‌خواهم بازپرداخت سفارشم را بگیرم"})
	if d.Topic != "Refund" || d.Action.Target != itoa(ids["carol"]) || !strings.Contains(d.Reason, "Refunds go to billing") {
		t.Fatalf("persian refund should reach the available billing specialist: %+v", d)
	}
	// Nobody in billing free: the job profile's escalation contact takes it.
	l[ids["carol"]] = domain.AgentPaused
	d = decide(t, ctx, e, domain.RouteRequest{Text: "I want a refund"})
	if d.Action.Target != itoa(ids["sara"]) || !strings.Contains(strings.Join(d.Steps, " "), "escalation contact") {
		t.Fatalf("expected escalation to the supervisor: %+v", d)
	}
	// Then the department queue.
	l[ids["sara"]] = domain.AgentOffline
	d = decide(t, ctx, e, domain.RouteRequest{Text: "I want a refund"})
	if d.Action.Type != domain.ActionQueue || !strings.Contains(d.TargetName, "Finance") {
		t.Fatalf("expected the Finance queue: %+v", d)
	}
	// Higher-priority dispute rule wins over the topic rule; its fallback is voicemail.
	d = decide(t, ctx, e, domain.RouteRequest{Text: "refund or I file a chargeback"})
	if d.Action.Type != domain.ActionVoicemail || d.Candidates[0].RuleName != "Chargebacks and disputes go to the supervisor" {
		t.Fatalf("dispute rule: %+v", d)
	}
}

func TestDecide_DraftRulesAreIgnored(t *testing.T) {
	ctx, e, ids, l := setup(t)
	l[ids["alice"]] = domain.AgentAvailable
	// The pricing rule is an unapproved AI draft: the Sales topic's
	// department queue is used instead of the sales representative.
	d := decide(t, ctx, e, domain.RouteRequest{Topic: "sales", Text: "how much is the pro plan"})
	if d.RuleID != 0 || d.Action.Type != domain.ActionQueue || !strings.Contains(d.TargetName, "Sales") {
		t.Fatalf("draft rule must not route: %+v", d)
	}
	d = decide(t, ctx, e, domain.RouteRequest{Text: "what's the weather"})
	if d.Action.Type != domain.ActionNone {
		t.Fatalf("unknown request should use the default handoff: %+v", d)
	}
}

func TestGenerate_TemplateDraftsNeedApproval(t *testing.T) {
	job := &domain.JobProfile{ID: 7, Title: "Billing specialist", RequestTypes: []string{"refund", "invoice copy"},
		EscalationConditions: "chargebacks; legal threats", EscalateToUserID: 9}
	rules, method := routing.Generate(context.Background(), nil, job, []domain.Topic{{Name: "Refund"}})
	if method != "template" || len(rules) != 4 {
		t.Fatalf("drafts: %s %+v", method, rules)
	}
	for _, r := range rules {
		if r.Status != domain.RuleDraft || r.Source != "ai" {
			t.Fatalf("generated rules must be drafts: %+v", r)
		}
	}
	if rules[0].Topics[0] != "Refund" || rules[2].TargetType != domain.TargetUser || rules[2].TargetID != 9 {
		t.Fatalf("mapping: %+v", rules)
	}
}

func itoa(v int64) string { return strconv.FormatInt(v, 10) }
