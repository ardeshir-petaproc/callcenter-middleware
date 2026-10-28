package core_test

import (
	"testing"

	"callcenter-middleware/internal/domain"
)

// connected puts a caller in a queue and connects them to the first agent.
func connected(t *testing.T, h *harness, q *domain.Queue, caller string, agents ...*domain.User) string {
	t.Helper()
	setupQueueFlow(h, q)
	for _, u := range agents {
		_ = h.core.SetAgentStatus(u.ID, domain.AgentAvailable, "")
	}
	h.incoming(caller, "+15550001", "200")
	h.dtmf(caller, "1")
	h.answer(h.sw.lastOrig().ChannelID)
	snap := h.core.Snapshot(0)
	if len(snap.Calls) != 1 || snap.Calls[0].State != domain.CallConnected {
		t.Fatalf("not connected: %+v", snap.Calls)
	}
	return snap.Calls[0].ID
}

func agentStatus(h *harness, id int64) domain.AgentStatus {
	st, _ := h.core.Agent(id)
	return st.Status
}

func TestWarmTransfer_ConsultCompleteAndCancel(t *testing.T) {
	h := newHarness(t, nil)
	alice, bob := h.agent("alice", "1001"), h.agent("bob", "1002")
	q := h.queue(domain.Queue{Name: "Sales"}, alice)
	callID := connected(t, h, q, "cust-1", alice, bob)

	// Consult bob: caller held, bob rings, then talks with alice.
	if err := h.core.Consult(callID, domain.Action{Type: domain.ActionAgent, Target: itoa(bob.ID)}); err != nil {
		t.Fatal(err)
	}
	consult := h.sw.lastOrig()
	if consult.Endpoint != "PJSIP/1002" || !h.sw.has("moh cust-1") || agentStatus(h, bob.ID) != domain.AgentRinging {
		t.Fatalf("consult start: %+v %v", consult, h.sw.ops)
	}
	// Bob does not answer: the caller is back with alice.
	h.gone(consult.ChannelID, 19)
	if v, _ := h.core.LiveCall(callID); v.State != domain.CallConnected || v.AgentID != alice.ID {
		t.Fatalf("expected return to alice: %+v", v.Call)
	}
	if !h.sw.has("stopmoh cust-1") || agentStatus(h, bob.ID) != domain.AgentAvailable {
		t.Fatalf("caller not resumed or bob stuck: %v %s", h.sw.ops, agentStatus(h, bob.ID))
	}

	// Second attempt: bob answers, alice completes the transfer.
	_ = h.core.Consult(callID, domain.Action{Type: domain.ActionAgent, Target: itoa(bob.ID)})
	h.answer(h.sw.lastOrig().ChannelID)
	if agentStatus(h, bob.ID) != domain.AgentBusy {
		t.Fatalf("bob should be busy consulting: %s", agentStatus(h, bob.ID))
	}
	if err := h.core.CompleteTransfer(callID); err != nil {
		t.Fatal(err)
	}
	v, _ := h.core.LiveCall(callID)
	if v.State != domain.CallConnected || v.AgentID != bob.ID || agentStatus(h, alice.ID) == domain.AgentBusy {
		t.Fatalf("after transfer: %+v alice=%s", v.Call, agentStatus(h, alice.ID))
	}
	if err := h.core.CancelConsult(callID); err == nil {
		t.Fatal("no consult should be active")
	}
}

func TestConference_AndAgentLeavesHandsOver(t *testing.T) {
	h := newHarness(t, nil)
	alice, bob := h.agent("alice", "1001"), h.agent("bob", "1002")
	q := h.queue(domain.Queue{Name: "Sales"}, alice)
	callID := connected(t, h, q, "cust-1", alice, bob)
	agentChan := h.sw.lastOrig().ChannelID

	_ = h.core.Consult(callID, domain.Action{Type: domain.ActionAgent, Target: itoa(bob.ID)})
	if err := h.core.Conference(callID); err == nil {
		t.Fatal("conference before the consult answers must fail")
	}
	h.answer(h.sw.lastOrig().ChannelID)
	if err := h.core.Conference(callID); err != nil {
		t.Fatal(err)
	}
	if v, _ := h.core.LiveCall(callID); v.State != domain.CallConnected {
		t.Fatalf("conference state: %+v", v.Call)
	}
	// Alice leaves the conference: bob keeps the caller.
	h.gone(agentChan, 16)
	v, ok := h.core.LiveCall(callID)
	if !ok || v.AgentID != bob.ID || v.State != domain.CallConnected {
		t.Fatalf("bob should keep the caller: %v %+v", ok, v.Call)
	}
}

func TestBlindTransferToAgent_ReturnsToPreviousOnNoAnswer(t *testing.T) {
	h := newHarness(t, nil)
	alice, bob := h.agent("alice", "1001"), h.agent("bob", "1002")
	q := h.queue(domain.Queue{Name: "Sales"}, alice)
	callID := connected(t, h, q, "cust-1", alice, bob)

	if err := h.core.Transfer(callID, domain.Action{Type: domain.ActionAgent, Target: itoa(bob.ID)}); err != nil {
		t.Fatal(err)
	}
	toBob := h.sw.lastOrig()
	if toBob.Endpoint != "PJSIP/1002" {
		t.Fatalf("expected bob to ring: %+v", toBob)
	}
	h.gone(toBob.ChannelID, 19)
	h.core.Tick()
	if back := h.sw.lastOrig(); back.Endpoint != "PJSIP/1001" {
		t.Fatalf("expected the call back at alice: %+v", back)
	}
}

func TestLinearStrategyAndRecordingNotice(t *testing.T) {
	h := newHarness(t, nil)
	alice, bob, carol := h.agent("alice", "1001"), h.agent("bob", "1002"), h.agent("carol", "1003")
	q := h.queue(domain.Queue{Name: "Seq", Strategy: domain.StrategyLinear, Record: true, RecordNotice: "sound:recorded"},
		carol, alice, bob)
	setupQueueFlow(h, q)
	for _, u := range []*domain.User{alice, bob, carol} {
		_ = h.core.SetAgentStatus(u.ID, domain.AgentAvailable, "")
	}
	h.incoming("cust-1", "+15550001", "200")
	h.dtmf("cust-1", "1")
	if !h.sw.has("play cust-1 sound:recorded") || h.sw.count("originate") != 0 {
		t.Fatalf("notice must play before any agent rings: %v", h.sw.ops)
	}
	h.event(domain.TelephonyEvent{Type: domain.EventPlaybackFinished, PlaybackID: h.sw.lastPlayID()})
	if first := h.sw.lastOrig(); first.Endpoint != "PJSIP/1003" {
		t.Fatalf("linear should ring carol (first member): %+v", first)
	}
	h.gone(h.sw.lastOrig().ChannelID, 19)
	h.core.Tick()
	if next := h.sw.lastOrig(); next.Endpoint != "PJSIP/1001" {
		t.Fatalf("then alice: %+v", next)
	}
}
