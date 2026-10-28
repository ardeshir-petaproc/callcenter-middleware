package core_test

import (
	"testing"

	"callcenter-middleware/internal/domain"
)

func TestAIRoute_ConnectsChosenEmployeeOrFallsBack(t *testing.T) {
	ai := &fakeAI{}
	h := newHarness(t, ai)
	alice, bob := h.agent("alice", "1001"), h.agent("bob", "1002")
	q := h.queue(domain.Queue{Name: "Humans"}, alice)
	p := &domain.AIProfile{Name: "Concierge", HandoffQueueID: q.ID, Enabled: true}
	_ = h.st.AIProfiles.Create(tctx(), p)
	_ = h.st.Flows.Create(tctx(), &domain.IVRFlow{Name: "ai-first", DNIS: "300", Enabled: true,
		DirectAction: domain.Action{Type: domain.ActionAI, Target: itoa(p.ID)}})
	h.reload()
	_ = h.core.SetAgentStatus(alice.ID, domain.AgentAvailable, "")
	_ = h.core.SetAgentStatus(bob.ID, domain.AgentAvailable, "")

	h.incoming("c1", "555", "300")
	callID := ai.started[0].CallID
	h.answer("media-" + callID)
	ai.ctl.AIRoute(callID, domain.Action{Type: domain.ActionAgent, Target: itoa(bob.ID)}, "Refund → Bob", "wants refund")
	h.core.Tick()
	if h.sw.lastOrig().Endpoint != "PJSIP/1002" {
		t.Fatalf("expected the routed employee to ring, got %+v", h.sw.lastOrig())
	}
	v, _ := h.core.LiveCall(callID)
	if v.Summary != "AI handoff: wants refund" {
		t.Fatalf("summary: %q", v.Summary)
	}

	// No decision: the profile's handoff queue takes the caller.
	h.incoming("c2", "556", "300")
	call2 := ai.started[1].CallID
	h.answer("media-" + call2)
	ai.ctl.AIRoute(call2, domain.Action{}, "no rule", "question")
	h.core.Tick()
	if h.sw.lastOrig().Endpoint != "PJSIP/1001" {
		t.Fatalf("expected handoff queue (alice), got %+v", h.sw.lastOrig())
	}

	// A rule fallback such as voicemail is executed as decided.
	h.incoming("c3", "557", "300")
	call3 := ai.started[2].CallID
	h.answer("media-" + call3)
	ai.ctl.AIRoute(call3, domain.Action{Type: domain.ActionVoicemail}, "fallback", "after hours")
	h.core.Tick()
	if !h.sw.has("play c3 sound:vm-intro") {
		t.Fatalf("expected voicemail, got %v", h.sw.ops)
	}
}
