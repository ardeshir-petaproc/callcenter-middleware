// Package integration runs the real ARI client, call core and HTTP API
// against the ARI simulator.
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"callcenter-middleware/internal/ari"
	"callcenter-middleware/internal/auth"
	"callcenter-middleware/internal/config"
	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/events"
	"callcenter-middleware/internal/joow"
	"callcenter-middleware/internal/kb"
	"callcenter-middleware/internal/mail"
	"callcenter-middleware/internal/retention"
	"callcenter-middleware/internal/routing"
	"callcenter-middleware/internal/secure"
	"callcenter-middleware/internal/seed"
	"callcenter-middleware/internal/server"
	"callcenter-middleware/internal/sim"
	"callcenter-middleware/internal/store"
	"callcenter-middleware/internal/tickets"
)

type stack struct {
	t    *testing.T
	sim  *sim.Sim
	api  *httptest.Server
	core *core.Core
	st   *store.Store
	idp  *joow.Local
}

func newStack(t *testing.T, speed float64) *stack {
	t.Helper()
	s := sim.New(sim.Options{App: "cc", Username: "u", Password: "p", Speed: speed, AnswerProb: 1, TrunkAnswer: 1})
	simSrv := httptest.NewServer(s.Handler())
	t.Cleanup(simSrv.Close)

	cfg := config.LoadFromEnv()
	cfg.ARIURL, cfg.ARIWSURL = simSrv.URL+"/ari", "ws"+strings.TrimPrefix(simSrv.URL, "http")+"/ari/events"
	cfg.ARIUsername, cfg.ARIPassword, cfg.ARIApp = "u", "p", "cc"
	cfg.DefaultWrapupSec = 0
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(func() { cancel(); st.Close() })
	tctx := store.ForTenant(ctx, core.DefaultTenant)
	if err := seed.EnsureAdmin(tctx, st, "admin", "admin123"); err != nil {
		t.Fatal(err)
	}
	if err := seed.Demo(tctx, st); err != nil {
		t.Fatal(err)
	}
	if err := seed.DemoSecondTenant(ctx, st); err != nil {
		t.Fatal(err)
	}
	idp := joow.NewLocal(st.DB(), "test")
	if err := seed.DemoJoowAccounts(ctx, idp.Upsert); err != nil {
		t.Fatal(err)
	}
	hub := events.NewHub()
	client := ari.NewClient(cfg.ARIURL, "u", "p", "cc", nil)
	stream := ari.NewStream(cfg.ARIWSURL, "cc", "u", "p", nil)
	c := core.New(core.Deps{Config: cfg, Switch: client, Store: st, Publisher: hub})
	evs := make(chan domain.TelephonyEvent, 256)
	go c.Run(ctx, evs)
	if err := c.Reload(ctx); err != nil {
		t.Fatal(err)
	}
	go func() { _ = stream.Subscribe(ctx, evs) }()
	box, tk := secure.New("test"), &tickets.Service{Store: st, Pub: hub, Joow: idp}
	kbs := &kb.Service{Store: st, Box: box, Dir: t.TempDir()}
	srv := server.NewServer("0", server.Deps{Config: cfg, Store: st, Core: c, Hub: hub,
		Issuer: auth.NewIssuer("test", time.Hour), Stream: stream, Fetcher: client, Started: time.Now(),
		Joow: idp, DevJoowTokens: true, Tickets: tk, KB: kbs,
		Mail:   &mail.Service{Store: st, Box: box, Tickets: tk, KB: kbs, Mailbox: mail.IMAP{Timeout: 5 * time.Second}},
		Router: &routing.Engine{Store: st, Live: c}, Retention: &retention.Service{Store: st, Deleter: client}})
	api := httptest.NewServer(srv.Handler())
	t.Cleanup(api.Close)
	waitFor(t, 5*time.Second, "ARI connected", stream.IsConnected)
	return &stack{t: t, sim: s, api: api, core: c, st: st, idp: idp}
}

func waitFor(t *testing.T, d time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func (s *stack) call(method, path, token string, body any, out any) int {
	s.t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, s.api.URL+"/api/v1"+path, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		s.t.Fatal(err)
	}
	defer resp.Body.Close()
	if out != nil {
		_ = json.NewDecoder(resp.Body).Decode(out)
	}
	return resp.StatusCode
}

func (s *stack) login(user, pass string) string {
	var res struct {
		Token string `json:"token"`
	}
	if code := s.call("POST", "/auth/login", "", map[string]string{"username": user, "password": pass}, &res); code != 200 {
		s.t.Fatalf("login %s: %d", user, code)
	}
	return res.Token
}

func TestStack_InboundCallAnsweredByAgent(t *testing.T) {
	s := newStack(t, 40)
	alice := s.login("alice", "agent123")
	if code := s.call("POST", "/me/status", alice, map[string]string{"status": "available"}, nil); code != 200 {
		t.Fatalf("set status: %d", code)
	}
	s.sim.InjectCall(sim.SimCall{Caller: "+15550100", DNIS: "100", Digits: "1", TalkSec: 60})

	admin := s.login("admin", "admin123")
	var snap core.Snapshot
	waitFor(t, 5*time.Second, "connected call", func() bool {
		s.call("GET", "/live", admin, nil, &snap)
		return len(snap.Calls) == 1 && snap.Calls[0].State == domain.CallConnected
	})
	live := snap.Calls[0]
	if live.CallerName != "Maria Rossi" || live.IVRPath != "1" {
		t.Fatalf("expected VIP contact match and IVR path, got %+v", live.Call)
	}
	var history struct {
		Items []domain.Call `json:"items"`
	}
	waitFor(t, 5*time.Second, "call ended", func() bool {
		s.call("GET", "/calls?outcome=answered&q=%2B15550100", admin, nil, &history)
		return len(history.Items) == 1
	})
	c := history.Items[0]
	if c.AgentID == 0 || c.Recording == "" || c.TalkSec <= 0 {
		t.Fatalf("unexpected CDR %+v", c)
	}
	// The recording proxy streams the simulator's WAV through the API.
	req, _ := http.NewRequest("GET", s.api.URL+"/api/v1/calls/"+c.ID+"/recording?token="+admin, nil)
	rec, err := http.DefaultClient.Do(req)
	if err != nil || rec.StatusCode != 200 || rec.Header.Get("Content-Type") != "audio/wav" {
		t.Fatalf("recording proxy failed: %v %v", err, rec)
	}
	rec.Body.Close()
	// The agent sees the call in their own history and can wrap it up.
	if code := s.call("PATCH", "/calls/"+c.ID, alice, map[string]any{"disposition": "sale", "notes": "upsold"}, nil); code != 200 {
		t.Fatalf("wrapup: %d", code)
	}
}
