package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"callcenter-middleware/internal/domain"
)

func open(t *testing.T) *Store {
	t.Helper()
	st, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestCollectionCRUD(t *testing.T) {
	st, ctx := open(t), ForTenant(context.Background(), 1)
	q := &domain.Queue{Name: "Sales", RequiredSkills: []string{"en"}, Overflow: domain.Action{Type: domain.ActionVoicemail}}
	if err := st.Queues.Create(ctx, q); err != nil || q.ID == 0 {
		t.Fatalf("create: %v %d", err, q.ID)
	}
	q.Name = "Sales EU"
	if err := st.Queues.Update(ctx, q); err != nil {
		t.Fatal(err)
	}
	got, err := st.Queues.Get(ctx, q.ID)
	if err != nil || got.Name != "Sales EU" || got.Overflow.Type != domain.ActionVoicemail {
		t.Fatalf("get: %v %+v", err, got)
	}
	if _, err := st.Flows.Get(ctx, q.ID); !errors.Is(err, ErrNotFound) {
		t.Fatal("collections must be isolated by kind")
	}
	if err := st.Queues.Delete(ctx, q.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.Queues.Delete(ctx, q.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestUsersUniqueAndPasswordKept(t *testing.T) {
	st, ctx := open(t), ForTenant(context.Background(), 1)
	u := &domain.User{Username: "Ana", Role: domain.RoleAgent, PasswordHash: "h1", Enabled: true}
	if err := st.CreateUser(ctx, u); err != nil {
		t.Fatal(err)
	}
	if err := st.CreateUser(ctx, &domain.User{Username: "Ana", Role: domain.RoleAgent, PasswordHash: "x"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
	u.PasswordHash, u.DisplayName = "", "Ana B"
	if err := st.UpdateUser(ctx, u); err != nil {
		t.Fatal(err)
	}
	got, _ := st.GetUserByUsername(ctx, "ana")
	if got.PasswordHash != "h1" || got.DisplayName != "Ana B" {
		t.Fatalf("unexpected user %+v", got)
	}
}

func TestReports(t *testing.T) {
	st, ctx := open(t), ForTenant(context.Background(), 1)
	base := time.Date(2026, 10, 20, 10, 0, 0, 0, time.UTC)
	mk := func(id string, answered bool, wait, talk int, sla bool, outcome domain.CallOutcome, agent int64) {
		c := &domain.Call{ID: id, Direction: "inbound", State: domain.CallEnded, Outcome: outcome, QueueID: 1,
			AgentID: agent, StartedAt: base, QueuedAt: &base, WaitSec: wait, TalkSec: talk, SLAMet: sla}
		if answered {
			a := base.Add(time.Duration(wait) * time.Second)
			c.AnsweredAt = &a
		}
		if err := st.SaveCall(ctx, c); err != nil {
			t.Fatal(err)
		}
	}
	mk("a", true, 10, 100, true, domain.OutcomeAnswered, 7)
	mk("b", true, 30, 200, false, domain.OutcomeAnswered, 7)
	mk("c", false, 50, 0, false, domain.OutcomeAbandoned, 0)
	mk("d", false, 5, 0, false, domain.OutcomeAbandoned, 0)

	sum, err := st.ReportSummary(ctx, domain.CallFilter{From: base.Add(-time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	if sum.Total != 4 || sum.Answered != 2 || sum.Abandoned != 2 || sum.AbandonRate != 50 ||
		sum.SLAPercent != 25 || sum.AvgWaitSec != 20 || sum.AvgTalkSec != 150 {
		t.Fatalf("unexpected summary %+v", sum)
	}
	agents, _ := st.ReportByGroup(ctx, domain.CallFilter{}, "agent_id")
	if len(agents) != 1 || agents[0].ID != 7 || agents[0].Answered != 2 {
		t.Fatalf("unexpected agent stats %+v", agents)
	}
	hours, _ := st.ReportBuckets(ctx, domain.CallFilter{}, "hour", 3600)
	if len(hours) != 1 || hours[0].Key != "11" || hours[0].Total != 4 {
		t.Fatalf("unexpected buckets %+v", hours)
	}
	byOutcome, _ := st.ReportCounts(ctx, domain.CallFilter{}, "outcome")
	if len(byOutcome) != 2 || byOutcome[0].Count != 2 {
		t.Fatalf("unexpected breakdown %+v", byOutcome)
	}
}

func TestAgentTime(t *testing.T) {
	st, ctx := open(t), ForTenant(context.Background(), 1)
	t0 := time.Date(2026, 10, 20, 9, 0, 0, 0, time.UTC)
	for _, e := range []domain.AgentEvent{
		{UserID: 1, Status: domain.AgentAvailable, At: t0.Add(-time.Hour)}, // before window
		{UserID: 1, Status: domain.AgentBusy, At: t0.Add(10 * time.Minute)},
		{UserID: 1, Status: domain.AgentPaused, At: t0.Add(30 * time.Minute)},
	} {
		if err := st.AddAgentEvent(ctx, e); err != nil {
			t.Fatal(err)
		}
	}
	out, err := st.ReportAgentTime(ctx, t0, t0.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Seconds["available"] != 600 || out[0].Seconds["busy"] != 1200 || out[0].Seconds["paused"] != 1800 {
		t.Fatalf("unexpected agent time %+v", out)
	}
}
