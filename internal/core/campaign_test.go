package core_test

import (
	"context"
	"testing"
	"time"

	"callcenter-middleware/internal/domain"
)

func TestSurvey_CollectsCSATAfterAgentHangsUp(t *testing.T) {
	h := newHarness(t, nil)
	a := h.agent("a", "1001")
	q := h.queue(domain.Queue{Name: "Care", Survey: true}, a)
	setupQueueFlow(h, q)
	_ = h.core.SetAgentStatus(a.ID, domain.AgentAvailable, "")
	h.incoming("c1", "555", "200")
	h.dtmf("c1", "1")
	agentLeg := h.sw.lastOrig()
	h.answer(agentLeg.ChannelID)
	h.advance(40 * time.Second)
	h.gone(agentLeg.ChannelID, 16) // agent hangs up first
	if h.sw.has("hangup c1") {
		t.Fatal("caller must stay on the line for the survey")
	}
	if st, _ := h.core.Agent(a.ID); st.Status == domain.AgentBusy {
		t.Fatal("agent should be released during the survey")
	}
	h.finishPlayback() // survey prompt done
	h.dtmf("c1", "4")
	h.finishPlayback() // thanks prompt done
	calls, _, _ := h.st.ListCalls(context.Background(), domain.CallFilter{})
	if len(calls) != 1 || calls[0].CSAT != 4 || calls[0].TalkSec != 40 || calls[0].Outcome != domain.OutcomeAnswered {
		t.Fatalf("unexpected CDR %+v", calls)
	}
}

func TestCampaign_ProgressiveDialerAndRetries(t *testing.T) {
	h := newHarness(t, nil)
	a := h.agent("a", "1001")
	q := h.queue(domain.Queue{Name: "Outbound"}, a)
	ctx := context.Background()
	cp := &domain.Campaign{Name: "Renewals", QueueID: q.ID, Status: domain.CampaignRunning, MaxAttempts: 2, RetryMinutes: 10}
	_ = h.st.Campaigns.Create(ctx, cp)
	_, _ = h.st.AddLeads(ctx, cp.ID, []domain.Lead{{Number: "+15550001", Name: "Lead One"}})
	h.reload()
	_ = h.core.SetAgentStatus(a.ID, domain.AgentAvailable, "")

	h.advance(3 * time.Second)
	agentLeg := h.sw.lastOrig()
	if agentLeg.Endpoint != "PJSIP/1001" {
		t.Fatalf("expected agent rung for campaign, got %+v", agentLeg)
	}
	h.answer(agentLeg.ChannelID)
	cust := h.sw.lastOrig()
	h.gone(cust.ChannelID, 17) // busy
	leads, _ := h.st.ListLeads(ctx, cp.ID, "", 10, 0)
	if leads[0].Status != domain.LeadPending || leads[0].Attempts != 1 {
		t.Fatalf("expected retry pending, got %+v", leads[0])
	}
	// Not due yet: nothing dialed.
	n := h.sw.count("originate")
	h.advance(3 * time.Second)
	if h.sw.count("originate") != n {
		t.Fatal("lead dialed before its retry time")
	}
	h.advance(10 * time.Minute)
	agentLeg = h.sw.lastOrig()
	h.answer(agentLeg.ChannelID)
	h.answer(h.sw.lastOrig().ChannelID) // customer answers
	h.advance(20 * time.Second)
	h.gone(agentLeg.ChannelID, 16)
	leads, _ = h.st.ListLeads(ctx, cp.ID, "", 10, 0)
	if leads[0].Status != domain.LeadAnswered || leads[0].Attempts != 2 {
		t.Fatalf("expected answered lead, got %+v", leads[0])
	}
	h.advance(3 * time.Second)
	got, _ := h.st.Campaigns.Get(ctx, cp.ID)
	if got.Status != domain.CampaignCompleted {
		t.Fatalf("expected campaign completed, got %s", got.Status)
	}
	calls, _, _ := h.st.ListCalls(ctx, domain.CallFilter{CampaignID: cp.ID})
	if len(calls) != 2 || calls[0].CallerName != "Lead One" {
		t.Fatalf("expected 2 campaign calls, got %+v", calls)
	}
}
