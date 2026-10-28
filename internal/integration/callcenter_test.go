package integration

import (
	"testing"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/sim"
)

type liveSnap struct {
	Calls []struct {
		domain.Call
		Consult string `json:"consult"`
	} `json:"calls"`
}

func TestWarmTransferQAAndRetention(t *testing.T) {
	s := newStack(t, 40)
	admin, reza := s.loginTenant("admin", "admin123", "parsian"), s.loginTenant("reza", "agent123", "parsian")
	bob := s.loginTenant("bob", "agent123", "parsian")
	s.call("POST", "/me/status", reza, map[string]string{"status": "available"}, nil)
	s.sim.InjectCall(sim.SimCall{Caller: "+15559990001", DNIS: "200", TalkSec: 600})
	var snap liveSnap
	waitFor(t, 5*time.Second, "call connected to reza", func() bool {
		s.call("GET", "/live", admin, nil, &snap)
		return len(snap.Calls) == 1 && snap.Calls[0].State == domain.CallConnected
	})
	call := snap.Calls[0]
	var me struct {
		User domain.User `json:"user"`
	}
	s.call("GET", "/auth/me", bob, nil, &me)
	s.call("POST", "/me/status", bob, map[string]string{"status": "available"}, nil)

	// Reza consults Bob while the caller waits on hold, then hands over.
	var cerr map[string]any
	if code := s.call("POST", "/calls/"+call.ID+"/consult", reza, domain.Action{Type: "agent", Target: itoa(me.User.ID)}, &cerr); code != 200 {
		t.Fatalf("consult: %d %v", code, cerr)
	}
	waitFor(t, 5*time.Second, "consult answered", func() bool {
		s.call("GET", "/live", admin, nil, &snap)
		return len(snap.Calls) == 1 && snap.Calls[0].Consult == "talking" && snap.Calls[0].State == domain.CallHold
	})
	if code := s.call("POST", "/calls/"+call.ID+"/consult/complete", reza, nil, nil); code != 200 {
		t.Fatalf("complete: %d", code)
	}
	s.call("GET", "/live", admin, nil, &snap)
	if len(snap.Calls) != 1 || snap.Calls[0].AgentID != me.User.ID || snap.Calls[0].State != domain.CallConnected {
		t.Fatalf("bob should now have the caller: %+v", snap.Calls)
	}
	s.call("POST", "/calls/"+call.ID+"/hangup", admin, nil, nil)

	// QA: score the call; the agent sees it only once final.
	var sc domain.QAScorecard
	if code := s.call("POST", "/qa/scorecards", admin, map[string]any{"name": "Standard", "pass_score": 70, "criteria": []map[string]any{
		{"label": "Greeting", "weight": 1, "max_score": 5}, {"label": "Verified identity", "weight": 2, "max_score": 1, "critical": true},
	}}, &sc); code != 201 && code != 200 {
		t.Fatalf("scorecard: %d", code)
	}
	var ev domain.QAEvaluation
	s.call("POST", "/qa/evaluations", admin, map[string]any{"call_id": call.ID, "scorecard_id": sc.ID,
		"scores": map[string]int{"greeting": 4, "verified_identity": 1}}, &ev)
	if ev.Score != 93.3 || !ev.Passed || ev.AgentID != me.User.ID || ev.Status != "draft" {
		t.Fatalf("evaluation: %+v", ev)
	}
	var mine []domain.QAEvaluation
	s.call("GET", "/qa/evaluations", bob, nil, &mine)
	if len(mine) != 0 {
		t.Fatalf("draft visible to agent: %+v", mine)
	}
	ev.Status, ev.Scores["verified_identity"] = "final", 0
	s.call("PUT", "/qa/evaluations/"+itoa(ev.ID), admin, ev, &ev)
	if !ev.AutoFailed || ev.Passed {
		t.Fatalf("critical miss must fail: %+v", ev)
	}
	if code := s.call("POST", "/qa/evaluations/"+itoa(ev.ID)+"/acknowledge", bob, map[string]string{"comment": "noted"}, &ev); code != 200 || !ev.Acknowledged {
		t.Fatalf("acknowledge: %d %+v", code, ev)
	}
	if code := s.call("POST", "/qa/evaluations", bob, map[string]any{"call_id": call.ID, "scorecard_id": sc.ID}, nil); code != 403 {
		t.Fatalf("agents cannot evaluate: %d", code)
	}

	// Recording downloads need recordings.export and are audited.
	if code := s.call("GET", "/calls/"+call.ID+"/recording?download=1", bob, nil, nil); code != 403 {
		t.Fatalf("agent download: %d", code)
	}
	s.call("GET", "/calls/"+call.ID+"/recording?download=1", admin, nil, nil)
	var audit []domain.AuditEntry
	s.call("GET", "/audit?action=recording.download", admin, nil, &audit)
	if len(audit) != 1 {
		t.Fatalf("download audit: %+v", audit)
	}
	// Retention policy round-trip and on-demand run.
	var pol map[string]int
	s.call("PUT", "/retention", admin, map[string]int{"recording_days": 30, "transcript_days": 90}, &pol)
	s.call("GET", "/retention", admin, nil, &pol)
	if pol["recording_days"] != 30 || pol["transcript_days"] != 90 {
		t.Fatalf("policy: %+v", pol)
	}
	if code := s.call("POST", "/retention/run", admin, nil, nil); code != 200 {
		t.Fatalf("run: %d", code)
	}
	if code := s.call("PUT", "/retention", bob, map[string]int{"recording_days": 1}, nil); code != 403 {
		t.Fatalf("agent policy change: %d", code)
	}
}
