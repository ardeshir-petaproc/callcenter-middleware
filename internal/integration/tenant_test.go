package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/sim"

	"github.com/coder/websocket"
)

func (s *stack) loginTenant(user, pass, tenant string) string {
	var res struct {
		Token string `json:"token"`
	}
	body := map[string]string{"username": user, "password": pass, "tenant": tenant}
	if code := s.call("POST", "/auth/login", "", body, &res); code != 200 {
		s.t.Fatalf("login %s@%s: %d", user, tenant, code)
	}
	return res.Token
}

func TestTenants_DataAndLiveStateAreIsolated(t *testing.T) {
	s := newStack(t, 40)
	acme := s.login("admin", "admin123")
	shop := s.loginTenant("admin", "admin123", "parsian")

	var acmeQueues, shopQueues []domain.Queue
	s.call("GET", "/queues", acme, nil, &acmeQueues)
	s.call("GET", "/queues", shop, nil, &shopQueues)
	if len(acmeQueues) != 3 || len(shopQueues) != 1 {
		t.Fatalf("expected 3 and 1 queues, got %d and %d", len(acmeQueues), len(shopQueues))
	}
	foreign := acmeQueues[0]
	for _, method := range []string{"GET", "PUT", "DELETE"} {
		if code := s.call(method, "/queues/"+itoa(foreign.ID), shop, foreign, nil); code != 404 {
			t.Fatalf("%s on another tenant's queue must 404, got %d", method, code)
		}
	}
	// A call to the shop's number is routed inside the shop only.
	reza := s.loginTenant("reza", "agent123", "parsian")
	s.call("POST", "/me/status", reza, map[string]string{"status": "available"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ws, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(s.api.URL, "http")+"/api/v1/ws?token="+acme, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Close(websocket.StatusNormalClosure, "")
	s.sim.InjectCall(sim.SimCall{Caller: "+15559990000", DNIS: "200", TalkSec: 30})
	var snap struct {
		Calls []domain.Call `json:"calls"`
	}
	waitFor(t, 5*time.Second, "shop call connected", func() bool {
		s.call("GET", "/live", shop, nil, &snap)
		return len(snap.Calls) == 1 && snap.Calls[0].State == domain.CallConnected
	})
	shopCall := snap.Calls[0]
	if shopCall.TenantID == 1 || shopCall.QueueID != shopQueues[0].ID {
		t.Fatalf("call routed outside the shop: %+v", shopCall)
	}
	s.call("GET", "/live", acme, nil, &snap)
	if len(snap.Calls) != 0 {
		t.Fatalf("acme sees the shop's live call: %+v", snap.Calls)
	}
	if code := s.call("GET", "/calls/"+shopCall.ID, acme, nil, nil); code != 404 {
		t.Fatalf("acme must not read the shop's call, got %d", code)
	}
	if code := s.call("POST", "/calls/"+shopCall.ID+"/hangup", acme, nil, nil); code != 404 {
		t.Fatalf("acme must not control the shop's call, got %d", code)
	}
	// Acme's realtime stream never carries the shop's events.
	readCtx, cancelRead := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancelRead()
	for {
		_, raw, err := ws.Read(readCtx)
		if err != nil {
			break
		}
		if strings.Contains(string(raw), shopCall.ID) {
			t.Fatalf("leaked event to another tenant: %s", raw)
		}
	}
}

func TestJoow_SingleSignOnAndTenantSwitch(t *testing.T) {
	s := newStack(t, 20)
	joowToken := func(acct string) string {
		var res struct {
			Token string `json:"token"`
		}
		s.call("POST", "/dev/joow-token", "", map[string]string{"account_id": acct}, &res)
		return res.Token
	}
	// Alice works for one organization: signed straight in.
	var session struct {
		Token  string        `json:"token"`
		User   domain.User   `json:"user"`
		Tenant domain.Tenant `json:"tenant"`
	}
	if code := s.call("POST", "/auth/joow", "", map[string]string{"token": joowToken("j-alice")}, &session); code != 200 || session.User.Username != "alice" {
		t.Fatalf("alice joow login: %d %+v", code, session.User)
	}
	// Bob works for two: he must choose, then can switch.
	var choice struct {
		Choose  bool `json:"choose_tenant"`
		Tenants []struct {
			ID   int64  `json:"id"`
			Slug string `json:"slug"`
		} `json:"tenants"`
	}
	bobJoow := joowToken("j-bob")
	s.call("POST", "/auth/joow", "", map[string]string{"token": bobJoow}, &choice)
	if !choice.Choose || len(choice.Tenants) != 2 {
		t.Fatalf("expected a choice of 2 organizations, got %+v", choice)
	}
	s.call("POST", "/auth/joow", "", map[string]any{"token": bobJoow, "tenant_id": choice.Tenants[0].ID}, &session)
	if session.Tenant.ID != choice.Tenants[0].ID {
		t.Fatalf("wrong tenant %+v", session.Tenant)
	}
	other := choice.Tenants[1].ID
	s.call("POST", "/auth/switch", session.Token, map[string]int64{"tenant_id": other}, &session)
	if session.Tenant.ID != other {
		t.Fatalf("switch failed: %+v", session.Tenant)
	}
	// Nima has a Joow account but is not an employee anywhere.
	if code := s.call("POST", "/auth/joow", "", map[string]string{"token": joowToken("j-nima")}, nil); code != 403 {
		t.Fatalf("non-member must be refused, got %d", code)
	}
	if code := s.call("POST", "/auth/joow", "", map[string]string{"token": "jdev.j-alice.forged"}, nil); code != 401 {
		t.Fatalf("forged Joow token must be rejected, got %d", code)
	}

	// The admin invites Nima; accepting with his Joow session creates the employee.
	admin := s.login("admin", "admin123")
	var inv struct {
		Token string `json:"token"`
	}
	if code := s.call("POST", "/invitations", admin, map[string]any{"joow_account_id": "j-nima", "role": "agent", "name": "Nima"}, &inv); code != 201 {
		t.Fatalf("invite: %d", code)
	}
	if code := s.call("POST", "/auth/accept-invite", "", map[string]string{"token": inv.Token, "joow_token": joowToken("j-alice")}, nil); code != 403 {
		t.Fatalf("invitation must be bound to its Joow account, got %d", code)
	}
	if code := s.call("POST", "/auth/accept-invite", "", map[string]string{"token": inv.Token, "joow_token": joowToken("j-nima")}, &session); code != 200 || session.User.JoowAccountID != "j-nima" {
		t.Fatalf("accept: %d %+v", code, session.User)
	}
	if code := s.call("POST", "/auth/accept-invite", "", map[string]string{"token": inv.Token, "joow_token": joowToken("j-nima")}, nil); code != 404 {
		t.Fatalf("an invitation can be used once, got %d", code)
	}
}
