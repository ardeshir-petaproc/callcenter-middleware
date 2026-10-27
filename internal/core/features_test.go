package core_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"
)

type fakeAI struct {
	mu      sync.Mutex
	started []core.AISpec
	stopped []string
	ctl     core.AIControl
}

func (f *fakeAI) Start(spec core.AISpec, ctl core.AIControl) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started, f.ctl = append(f.started, spec), ctl
	return "10.0.0.5:40000", nil
}
func (f *fakeAI) Stop(id string)           { f.mu.Lock(); f.stopped = append(f.stopped, id); f.mu.Unlock() }
func (f *fakeAI) Active() int              { return 0 }
func (f *fakeAI) SetRemote(string, string) {}

func TestAIAgent_BridgesMediaAndHandsOffToQueue(t *testing.T) {
	ai := &fakeAI{}
	h := newHarness(t, ai)
	alice := h.agent("alice", "1001")
	q := h.queue(domain.Queue{Name: "Humans"}, alice)
	p := &domain.AIProfile{Name: "Concierge", HandoffQueueID: q.ID, Enabled: true}
	_ = h.st.AIProfiles.Create(context.Background(), p)
	_ = h.st.Flows.Create(context.Background(), &domain.IVRFlow{Name: "ai-first", DNIS: "300", Enabled: true,
		DirectAction: domain.Action{Type: domain.ActionAI, Target: itoa(p.ID)}})
	h.reload()
	_ = h.core.SetAgentStatus(alice.ID, domain.AgentAvailable, "")

	h.incoming("c1", "555", "300")
	if len(ai.started) != 1 || !h.sw.has("extmedia media-") {
		t.Fatalf("expected AI session + external media, got %v", h.sw.ops)
	}
	callID := ai.started[0].CallID
	h.answer("media-" + callID)
	if h.sw.count("addbridge br-"+callID) != 2 {
		t.Fatalf("expected caller and media bridged, got %v", h.sw.ops)
	}
	ai.ctl.AITranscript(callID, "caller", "I need a human")
	ai.ctl.AITransfer(callID, "", "caller asked", "wants refund")
	h.core.Tick()
	if len(ai.stopped) == 0 || !h.sw.has("hangup media-"+callID) {
		t.Fatalf("expected AI torn down, got %v", h.sw.ops)
	}
	if h.sw.lastOrig().Endpoint != "PJSIP/1001" {
		t.Fatalf("expected handoff to ring alice, got %+v", h.sw.lastOrig())
	}
	v, _ := h.core.LiveCall(callID)
	if v.Summary != "AI handoff: wants refund" || v.AIProfileID != p.ID {
		t.Fatalf("unexpected call view %+v", v.Call)
	}
	tr, _ := h.st.Transcript(context.Background(), callID)
	if len(tr) != 1 || tr[0].Text != "I need a human" {
		t.Fatalf("unexpected transcript %+v", tr)
	}
}

func TestAfterHours_RunsAfterHoursAction(t *testing.T) {
	h := newHarness(t, nil)
	sch := &domain.Schedule{Name: "office", Weekly: []domain.TimeRange{{Day: 1, Start: "09:00", End: "17:00"}}}
	_ = h.st.Schedules.Create(context.Background(), sch)
	_ = h.st.Flows.Create(context.Background(), &domain.IVRFlow{Name: "main", DNIS: "*", Enabled: true,
		GreetingMedia: "sound:menu", ScheduleID: sch.ID, AfterHoursAction: domain.Action{Type: domain.ActionVoicemail}})
	h.reload()
	h.incoming("c1", "555", "200") // Tuesday: closed
	if !h.sw.has("play c1 sound:vm-intro") || h.sw.has("play c1 sound:menu") {
		t.Fatalf("expected voicemail instead of menu, got %v", h.sw.ops)
	}
}

func TestTransfer_ConnectedCallToAnotherQueue(t *testing.T) {
	h := newHarness(t, nil)
	a := h.agent("a", "1001")
	b := h.agent("b", "1002")
	q1 := h.queue(domain.Queue{Name: "L1"}, a)
	q2 := h.queue(domain.Queue{Name: "L2"}, b)
	setupQueueFlow(h, q1)
	_ = h.core.SetAgentStatus(a.ID, domain.AgentAvailable, "")
	_ = h.core.SetAgentStatus(b.ID, domain.AgentAvailable, "")
	h.incoming("c1", "555", "200")
	h.dtmf("c1", "1")
	legA := h.sw.lastOrig()
	h.answer(legA.ChannelID)
	callID := h.core.Snapshot().Calls[0].ID

	if err := h.core.Transfer(callID, domain.Action{Type: domain.ActionQueue, Target: itoa(q2.ID)}); err != nil {
		t.Fatal(err)
	}
	if !h.sw.has("hangup "+legA.ChannelID) || h.sw.lastOrig().Endpoint != "PJSIP/1002" {
		t.Fatalf("expected agent a released and b rung, got %v", h.sw.ops)
	}
	h.answer(h.sw.lastOrig().ChannelID)
	v, _ := h.core.LiveCall(callID)
	if v.AgentID != b.ID || v.QueueID != q2.ID || v.State != domain.CallConnected {
		t.Fatalf("unexpected call after transfer %+v", v.Call)
	}
}

func TestOutboundDial_AgentThenCustomer(t *testing.T) {
	h := newHarness(t, nil)
	a := h.agent("a", "1001")
	h.reload()
	callID, err := h.core.Dial(a.ID, "+1 555 0100", 0)
	if err != nil {
		t.Fatal(err)
	}
	agentLeg := h.sw.lastOrig()
	if agentLeg.Endpoint != "PJSIP/1001" {
		t.Fatalf("expected agent rung first, got %+v", agentLeg)
	}
	h.answer(agentLeg.ChannelID)
	cust := h.sw.lastOrig()
	if cust.Endpoint != "PJSIP/+15550100@trunk" || !h.sw.has("ring "+agentLeg.ChannelID) {
		t.Fatalf("expected customer dialed via trunk, got %+v %v", cust, h.sw.ops)
	}
	h.answer(cust.ChannelID)
	v, _ := h.core.LiveCall(callID)
	if v.State != domain.CallConnected || v.Direction != domain.DirectionOutbound {
		t.Fatalf("unexpected outbound call %+v", v.Call)
	}
	if err := h.core.Monitor(callID, a.ID, core.MonitorWhisper); err != nil {
		t.Fatal(err)
	}
	sup := h.sw.lastOrig()
	h.answer(sup.ChannelID)
	if !h.sw.has("snoop " + agentLeg.ChannelID) {
		t.Fatalf("expected snoop on agent leg, got %v", h.sw.ops)
	}
	h.advance(10 * time.Second)
	h.gone(agentLeg.ChannelID, 16)
	if !h.sw.has("hangup "+cust.ChannelID) || !h.sw.has("hangup "+sup.ChannelID) {
		t.Fatalf("expected all legs hung up, got %v", h.sw.ops)
	}
}
