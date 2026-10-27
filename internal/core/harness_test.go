package core_test

import (
	"callcenter-middleware/internal/config"
	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
	"context"
	"testing"
	"time"
)

// harness wires a core to a fake switch and an in-memory store.
type harness struct {
	t     *testing.T
	core  *core.Core
	sw    *fakeSwitch
	st    *store.Store
	clock time.Time
	stop  context.CancelFunc
}

func newHarness(t *testing.T, ai core.AIManager) *harness {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{
		DefaultQueue: "600", QueueContext: "ext-queues", IVRTimeout: 5 * time.Second, MaxRetries: 2,
		GreetingMedia: "sound:welcome", QueueMappings: map[string]string{"1": "600", "2": "601"},
		DefaultRingTimeout: 15, DefaultWrapupSec: 0, AutoPauseMisses: 2, VoicemailMaxSec: 60,
		VoicemailMedia: "sound:vm-intro", CallbackMedia: "sound:cb", OutboundDialTemplate: "PJSIP/{number}@trunk",
		DrainTimeout: time.Second,
	}
	sw := &fakeSwitch{}
	h := &harness{t: t, sw: sw, st: st, clock: time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC)}
	h.core = core.New(core.Deps{Config: cfg, Switch: sw, Store: st, AI: ai})
	h.core.SetClock(func() time.Time { return h.clock })
	ctx, cancel := context.WithCancel(context.Background())
	h.stop = cancel
	go h.core.Run(ctx, make(chan domain.TelephonyEvent))
	t.Cleanup(func() { cancel(); st.Close() })
	return h
}

func (h *harness) reload() {
	if err := h.core.Reload(context.Background()); err != nil {
		h.t.Fatal(err)
	}
}

func (h *harness) advance(d time.Duration) {
	h.clock = h.clock.Add(d)
	h.core.Tick()
}

func (h *harness) event(ev domain.TelephonyEvent) { h.core.Inject(ev) }

func (h *harness) incoming(ch, number, dnis string) {
	h.event(domain.TelephonyEvent{Type: domain.EventCallStarted, ChannelID: ch, CallerNumber: number, DNIS: dnis})
}

func (h *harness) dtmf(ch, d string) {
	h.event(domain.TelephonyEvent{Type: domain.EventDTMFReceived, ChannelID: ch, Digit: d})
}

func (h *harness) answer(ch string) {
	h.event(domain.TelephonyEvent{Type: domain.EventCallStarted, ChannelID: ch, Args: []string{"x"}})
}

func (h *harness) gone(ch string, cause int) {
	h.event(domain.TelephonyEvent{Type: domain.EventChannelDestroyed, ChannelID: ch, Cause: cause})
}

func (h *harness) agent(name, ext string) *domain.User {
	u := &domain.User{Username: name, Role: domain.RoleAgent, Extension: ext, Enabled: true, PasswordHash: "x"}
	if err := h.st.CreateUser(context.Background(), u); err != nil {
		h.t.Fatal(err)
	}
	return u
}

func (h *harness) queue(q domain.Queue, members ...*domain.User) *domain.Queue {
	q.Enabled = true
	if err := h.st.Queues.Create(context.Background(), &q); err != nil {
		h.t.Fatal(err)
	}
	var ms []domain.QueueMember
	for _, u := range members {
		ms = append(ms, domain.QueueMember{QueueID: q.ID, UserID: u.ID})
	}
	if err := h.st.SetQueueMembers(context.Background(), q.ID, ms); err != nil {
		h.t.Fatal(err)
	}
	return &q
}
