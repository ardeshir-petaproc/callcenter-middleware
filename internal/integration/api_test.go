package integration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"callcenter-middleware/internal/domain"

	"github.com/coder/websocket"
)

func TestAPI_AuthAndRoles(t *testing.T) {
	s := newStack(t, 20)
	if code := s.call("GET", "/live", "", nil, nil); code != 401 {
		t.Fatalf("expected 401 without token, got %d", code)
	}
	if code := s.call("POST", "/auth/login", "", map[string]string{"username": "admin", "password": "nope"}, nil); code != 401 {
		t.Fatalf("expected 401 for bad password, got %d", code)
	}
	agent := s.login("bob", "agent123")
	sup := s.login("sara", "agent123")
	admin := s.login("admin", "admin123")

	q := domain.Queue{Name: "VIP", Strategy: "ringall"}
	if code := s.call("POST", "/queues", agent, q, nil); code != 403 {
		t.Fatalf("agent must not create queues, got %d", code)
	}
	if code := s.call("POST", "/queues", sup, q, nil); code != 403 {
		t.Fatalf("supervisor must not create queues, got %d", code)
	}
	var created domain.Queue
	if code := s.call("POST", "/queues", admin, q, &created); code != 201 || created.ID == 0 || created.SLASec != 20 {
		t.Fatalf("admin create queue: %d %+v", code, created)
	}
	if code := s.call("POST", "/queues", admin, domain.Queue{Name: "x", Strategy: "bogus"}, nil); code != 400 {
		t.Fatalf("expected validation error, got %d", code)
	}
	if code := s.call("GET", "/reports/summary", agent, nil, nil); code != 403 {
		t.Fatalf("agent must not read reports, got %d", code)
	}
	if code := s.call("GET", "/reports/summary?from=2020-01-01", sup, nil, nil); code != 200 {
		t.Fatalf("supervisor reports: %d", code)
	}
	if code := s.call("GET", "/audit", sup, nil, nil); code != 403 {
		t.Fatalf("audit is admin only, got %d", code)
	}
	var audit []domain.AuditEntry
	s.call("GET", "/audit", admin, nil, &audit)
	if len(audit) == 0 {
		t.Fatal("expected audit entries")
	}
	// Agents may manage contacts (CRM) but only see their own calls.
	var ct domain.Contact
	if code := s.call("POST", "/contacts", agent, domain.Contact{Name: "New Lead", Phone: "+1 (555) 222-0000"}, &ct); code != 201 {
		t.Fatalf("agent create contact: %d", code)
	}
	var found struct {
		Contact *domain.Contact `json:"contact"`
	}
	s.call("GET", "/contacts/lookup?phone=15552220000", agent, nil, &found)
	if found.Contact == nil || found.Contact.ID != ct.ID {
		t.Fatalf("lookup by normalized phone failed: %+v", found)
	}
	var calls struct {
		Total int `json:"total"`
	}
	s.call("GET", "/calls", agent, nil, &calls)
	var all struct {
		Total int `json:"total"`
	}
	s.call("GET", "/calls", admin, nil, &all)
	if calls.Total >= all.Total {
		t.Fatalf("agent should see a subset of calls: %d vs %d", calls.Total, all.Total)
	}
}

func TestAPI_UserLifecycleAndMembership(t *testing.T) {
	s := newStack(t, 20)
	admin := s.login("admin", "admin123")
	var queues []domain.Queue
	s.call("GET", "/queues", admin, nil, &queues)
	var u domain.User
	body := map[string]any{"username": "erin", "password": "secret99", "role": "agent", "extension": "1009",
		"queues": []map[string]any{{"queue_id": queues[0].ID, "penalty": 2}}}
	if code := s.call("POST", "/users", admin, body, &u); code != 201 {
		t.Fatalf("create user: %d", code)
	}
	if code := s.call("POST", "/users", admin, body, nil); code != 409 {
		t.Fatalf("duplicate username should conflict, got %d", code)
	}
	var members []domain.QueueMember
	s.call("GET", "/members", admin, nil, &members)
	found := false
	for _, m := range members {
		found = found || (m.UserID == u.ID && m.QueueID == queues[0].ID && m.Penalty == 2)
	}
	if !found {
		t.Fatal("membership not stored")
	}
	erin := s.login("erin", "secret99")
	var st domain.AgentState
	if code := s.call("POST", "/me/status", erin, map[string]string{"status": "paused", "reason": "Break"}, &st); code != 200 || st.Status != domain.AgentPaused {
		t.Fatalf("pause: %d %+v", code, st)
	}
	if code := s.call("POST", "/me/status", erin, map[string]string{"status": "busy"}, nil); code != 400 {
		t.Fatalf("agents cannot set busy, got %d", code)
	}
	if code := s.call("DELETE", "/users/1", admin, nil, nil); code != 400 {
		t.Fatalf("admin cannot delete self, got %d", code)
	}
}

func TestAPI_WebSocketAndClickToCall(t *testing.T) {
	s := newStack(t, 40)
	alice := s.login("alice", "agent123")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(s.api.URL, "http")+"/api/v1/ws?token="+alice, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close(websocket.StatusNormalClosure, "")
	var first domain.Event
	if _, raw, err := ws.Read(ctx); err != nil || json.Unmarshal(raw, &first) != nil || first.Type != "snapshot" {
		t.Fatalf("expected snapshot first: %v %s", err, first.Type)
	}
	var res struct {
		CallID string `json:"call_id"`
	}
	if code := s.call("POST", "/me/dial", alice, map[string]string{"number": "+1 555 0101"}, &res); code != 200 || res.CallID == "" {
		t.Fatalf("dial: %d", code)
	}
	seen := map[string]bool{}
	for !seen["connected"] || !seen["ended"] {
		_, raw, err := ws.Read(ctx)
		if err != nil {
			t.Fatalf("ws read: %v (seen %v)", err, seen)
		}
		var ev struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		_ = json.Unmarshal(raw, &ev)
		var c domain.Call
		_ = json.Unmarshal(ev.Data, &c)
		if c.ID != res.CallID {
			continue
		}
		if ev.Type == domain.EvCallUpdated && c.State == domain.CallConnected {
			seen["connected"] = true
		}
		if ev.Type == domain.EvCallEnded {
			seen["ended"] = true
			if c.Direction != domain.DirectionOutbound || c.CallerName != "John Smith" {
				t.Fatalf("unexpected outbound CDR %+v", c)
			}
		}
	}
}
