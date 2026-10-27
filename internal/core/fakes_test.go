package core_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"callcenter-middleware/internal/config"
	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
)

// fakeSwitch records every ARI command issued by the core.
type fakeSwitch struct {
	mu         sync.Mutex
	ops        []string
	originated []domain.OriginateRequest
	failOrig   bool
}

func (f *fakeSwitch) rec(format string, args ...any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.ops = append(f.ops, fmt.Sprintf(format, args...))
	return nil
}

func (f *fakeSwitch) has(prefix string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, op := range f.ops {
		if strings.HasPrefix(op, prefix) {
			return true
		}
	}
	return false
}

func (f *fakeSwitch) count(prefix string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	n := 0
	for _, op := range f.ops {
		if strings.HasPrefix(op, prefix) {
			n++
		}
	}
	return n
}

func (f *fakeSwitch) lastOrig() domain.OriginateRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.originated) == 0 {
		return domain.OriginateRequest{}
	}
	return f.originated[len(f.originated)-1]
}

func (f *fakeSwitch) AnswerChannel(_ context.Context, ch string) error { return f.rec("answer %s", ch) }
func (f *fakeSwitch) PlayMedia(_ context.Context, ch, uri, id string) error {
	return f.rec("play %s %s %s", ch, uri, id)
}
func (f *fakeSwitch) StopPlayback(_ context.Context, id string) error { return f.rec("stopplay %s", id) }
func (f *fakeSwitch) ContinueToQueue(_ context.Context, ch, cx, ext string, p int) error {
	return f.rec("continue %s %s %s", ch, cx, ext)
}
func (f *fakeSwitch) HangupChannel(_ context.Context, ch, r string) error { return f.rec("hangup %s", ch) }
func (f *fakeSwitch) Originate(_ context.Context, r domain.OriginateRequest) error {
	if f.failOrig {
		return fmt.Errorf("originate failed")
	}
	f.mu.Lock()
	f.originated = append(f.originated, r)
	f.mu.Unlock()
	return f.rec("originate %s %s", r.Endpoint, r.ChannelID)
}
func (f *fakeSwitch) CreateBridge(_ context.Context, id, t string) error { return f.rec("bridge %s", id) }
func (f *fakeSwitch) DestroyBridge(_ context.Context, id string) error  { return f.rec("unbridge %s", id) }
func (f *fakeSwitch) AddToBridge(_ context.Context, id string, chans ...string) error {
	return f.rec("addbridge %s %s", id, strings.Join(chans, ","))
}
func (f *fakeSwitch) RemoveFromBridge(_ context.Context, id, ch string) error {
	return f.rec("rmbridge %s %s", id, ch)
}
func (f *fakeSwitch) StartMOH(_ context.Context, ch, class string) error { return f.rec("moh %s", ch) }
func (f *fakeSwitch) StopMOH(_ context.Context, ch string) error         { return f.rec("stopmoh %s", ch) }
func (f *fakeSwitch) StartRinging(_ context.Context, ch string) error    { return f.rec("ring %s", ch) }
func (f *fakeSwitch) StopRinging(_ context.Context, ch string) error     { return f.rec("stopring %s", ch) }
func (f *fakeSwitch) RecordBridge(_ context.Context, id, name string) error {
	return f.rec("recbridge %s %s", id, name)
}
func (f *fakeSwitch) RecordChannel(_ context.Context, ch, name string, max int) error {
	return f.rec("recchan %s %s", ch, name)
}
func (f *fakeSwitch) StopRecording(_ context.Context, name string) error { return f.rec("stoprec %s", name) }
func (f *fakeSwitch) ExternalMedia(_ context.Context, r domain.ExternalMediaRequest) error {
	return f.rec("extmedia %s %s", r.ChannelID, r.ExternalHost)
}
func (f *fakeSwitch) Snoop(_ context.Context, r domain.SnoopRequest) error {
	return f.rec("snoop %s %s %s", r.ChannelID, r.SnoopID, r.Whisper)
}
func (f *fakeSwitch) ListChannels(_ context.Context) ([]string, error) { return nil, nil }
func (f *fakeSwitch) SetMute(_ context.Context, ch string, m bool) error { return f.rec("mute %s %v", ch, m) }

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
