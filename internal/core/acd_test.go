package core_test

import (
	"testing"
	"time"

	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"
)

func setupQueueFlow(h *harness, q *domain.Queue) {
	flow := &domain.IVRFlow{Name: "main", DNIS: "*", GreetingMedia: "sound:menu", TimeoutSec: 5, MaxRetries: 1,
		Options: map[string]domain.Action{"1": {Type: domain.ActionQueue, Target: itoa(q.ID)}}, Enabled: true}
	if err := h.st.Flows.Create(tctx(), flow); err != nil {
		h.t.Fatal(err)
	}
	h.reload()
}

func itoa(v int64) string { return core.Itoa(v) }

func TestNativeQueue_RingAnswerBridgeAndWrapup(t *testing.T) {
	h := newHarness(t, nil)
	alice := h.agent("alice", "1001")
	q := h.queue(domain.Queue{Name: "Sales", Strategy: domain.StrategyLongestIdle, WrapupSec: 5, Record: true}, alice)
	setupQueueFlow(h, q)
	_ = h.core.SetAgentStatus(alice.ID, domain.AgentAvailable, "")

	h.incoming("cust-1", "+15550001", "200")
	h.dtmf("cust-1", "1")
	if !h.sw.has("moh cust-1") {
		t.Fatalf("expected MOH, got %v", h.sw.ops)
	}
	orig := h.sw.lastOrig()
	if orig.Endpoint != "PJSIP/1001" {
		t.Fatalf("expected ring to alice, got %+v", orig)
	}
	if st, _ := h.core.Agent(alice.ID); st.Status != domain.AgentRinging {
		t.Fatalf("expected ringing, got %s", st.Status)
	}

	h.answer(orig.ChannelID)
	if !h.sw.has("addbridge br-") || !h.sw.has("recbridge") {
		t.Fatalf("expected bridge+recording, got %v", h.sw.ops)
	}
	snap := h.core.Snapshot(0)
	if len(snap.Calls) != 1 || snap.Calls[0].State != domain.CallConnected || snap.Calls[0].AgentID != alice.ID {
		t.Fatalf("unexpected snapshot %+v", snap.Calls)
	}

	h.advance(30 * time.Second)
	h.gone("cust-1", 16)
	if !h.sw.has("hangup " + orig.ChannelID) {
		t.Fatalf("expected agent leg hung up, got %v", h.sw.ops)
	}
	st, _ := h.core.Agent(alice.ID)
	if st.Status != domain.AgentWrapup || st.CallsToday != 1 {
		t.Fatalf("expected wrapup after call, got %+v", st)
	}
	h.advance(6 * time.Second)
	if st, _ := h.core.Agent(alice.ID); st.Status != domain.AgentAvailable {
		t.Fatalf("expected available after wrapup, got %s", st.Status)
	}
	calls, _, _ := h.st.ListCalls(tctx(), domain.CallFilter{})
	c := calls[0]
	if c.Outcome != domain.OutcomeAnswered || c.TalkSec != 30 || c.AgentID != alice.ID || !c.SLAMet {
		t.Fatalf("unexpected CDR %+v", c)
	}
}

func TestNativeQueue_NoAnswerRollsToNextAgentAndAutoPauses(t *testing.T) {
	h := newHarness(t, nil)
	a := h.agent("a", "1001")
	b := h.agent("b", "1002")
	q := h.queue(domain.Queue{Name: "Support", Strategy: domain.StrategyRoundRobin}, a, b)
	setupQueueFlow(h, q)
	_ = h.core.SetAgentStatus(a.ID, domain.AgentAvailable, "")
	_ = h.core.SetAgentStatus(b.ID, domain.AgentAvailable, "")

	for i, ch := range []string{"c1", "c2"} {
		h.incoming(ch, "555", "200")
		h.dtmf(ch, "1")
		first := h.sw.lastOrig()
		if first.Endpoint != "PJSIP/1001" {
			t.Fatalf("call %d: expected agent a first, got %s", i, first.Endpoint)
		}
		h.gone(first.ChannelID, 19) // no answer
		second := h.sw.lastOrig()
		if second.Endpoint != "PJSIP/1002" {
			t.Fatalf("call %d: expected roll to b, got %s", i, second.Endpoint)
		}
		h.answer(second.ChannelID)
		h.gone(ch, 16)
	}
	st, _ := h.core.Agent(a.ID)
	if st.Status != domain.AgentPaused || st.PauseReason != "Missed calls" {
		t.Fatalf("expected a auto-paused after 2 misses, got %+v", st)
	}
}

func TestNativeQueue_AbandonAndOverflowToVoicemail(t *testing.T) {
	h := newHarness(t, nil)
	q := h.queue(domain.Queue{Name: "Billing", MaxWaitSec: 30})
	setupQueueFlow(h, q)

	h.incoming("c1", "555", "200")
	h.dtmf("c1", "1")
	h.advance(31 * time.Second)
	if !h.sw.has("play c1 sound:vm-intro") {
		t.Fatalf("expected voicemail prompt after overflow, got %v", h.sw.ops)
	}
	h.finishPlayback()
	if !h.sw.has("recchan c1 vm-") {
		t.Fatalf("expected voicemail recording, got %v", h.sw.ops)
	}
	snap := h.core.Snapshot(0)
	name := "vm-" + snap.Calls[0].ID
	h.event(domain.TelephonyEvent{Type: domain.EventRecordingFinished, RecordingName: name, DurationSec: 7})
	vms, _ := h.st.ListVoicemails(tctx(), 10)
	if len(vms) != 1 || vms[0].DurationSec != 7 || vms[0].CallerNumber != "555" {
		t.Fatalf("unexpected voicemails %+v", vms)
	}
	calls, _, _ := h.st.ListCalls(tctx(), domain.CallFilter{})
	if calls[0].Outcome != domain.OutcomeVoicemail {
		t.Fatalf("expected voicemail outcome, got %s", calls[0].Outcome)
	}

	// A second caller hangs up while waiting: abandoned.
	h.incoming("c2", "556", "200")
	h.dtmf("c2", "1")
	h.advance(5 * time.Second)
	h.gone("c2", 16)
	calls, _, _ = h.st.ListCalls(tctx(), domain.CallFilter{Outcome: string(domain.OutcomeAbandoned)})
	if len(calls) != 1 || calls[0].WaitSec != 5 {
		t.Fatalf("expected abandoned call with 5s wait, got %+v", calls)
	}
}

func TestQueue_CallbackRequestFromQueue(t *testing.T) {
	h := newHarness(t, nil)
	q := h.queue(domain.Queue{Name: "Sales", AllowCallback: true})
	setupQueueFlow(h, q)
	h.incoming("c1", "+15551234", "200")
	h.dtmf("c1", "1")
	h.dtmf("c1", "*")
	cbs, _ := h.st.ListCallbacks(tctx(), "", 10)
	if len(cbs) != 1 || cbs[0].Number != "+15551234" || cbs[0].QueueID != q.ID {
		t.Fatalf("unexpected callbacks %+v", cbs)
	}
	h.finishPlayback()
	if !h.sw.has("hangup c1") {
		t.Fatalf("expected hangup after confirmation, got %v", h.sw.ops)
	}
}
