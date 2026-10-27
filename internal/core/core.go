// Package core is the call control brain of the middleware. All live call,
// queue and agent state is owned by a single goroutine (the loop); ARI
// events, API commands, AI callbacks and timers are serialized through it,
// which keeps state transitions race-free without fine-grained locking.
package core

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"callcenter-middleware/internal/config"
	"callcenter-middleware/internal/domain"
)

// ErrStopped is returned when a command is sent after the loop exited.
var ErrStopped = errors.New("call core stopped")

// Core is the call manager.
type Core struct {
	Deps
	cfg  *config.Config
	cmds chan func()
	done chan struct{}
	now  func() time.Time
	ctx  context.Context

	calls      map[string]*liveCall
	legs       map[string]*leg
	playbacks  map[string]string
	recordings map[string]recordingRef
	agents     map[int64]*agentRT
	cache      *configCache
	rr         map[int64]int
	seq        int
	lastDial   time.Time
}

// New creates a core; call Reload before Run to load configuration.
func New(d Deps) *Core {
	return &Core{
		Deps:       d,
		cfg:        d.Config,
		cmds:       make(chan func(), 256),
		done:       make(chan struct{}),
		now:        time.Now,
		ctx:        context.Background(),
		calls:      map[string]*liveCall{},
		legs:       map[string]*leg{},
		playbacks:  map[string]string{},
		recordings: map[string]recordingRef{},
		agents:     map[int64]*agentRT{},
		cache:      &configCache{},
		rr:         map[int64]int{},
	}
}

// Run processes events and commands until ctx is cancelled.
func (c *Core) Run(ctx context.Context, events <-chan domain.TelephonyEvent) {
	defer close(c.done)
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			c.drainCommands()
			return
		case ev, ok := <-events:
			if !ok {
				events = nil
				continue
			}
			c.safe(func() { c.handleEvent(ev) })
		case fn := <-c.cmds:
			c.safe(fn)
		case <-ticker.C:
			c.safe(c.tick)
		}
	}
}

func (c *Core) drainCommands() {
	for {
		select {
		case fn := <-c.cmds:
			c.safe(fn)
		default:
			return
		}
	}
}

func (c *Core) safe(fn func()) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Call core recovered from panic", "panic", r)
		}
	}()
	fn()
}

// post schedules fn on the loop without waiting.
func (c *Core) post(fn func()) {
	select {
	case c.cmds <- fn:
	case <-c.done:
	}
}

// do runs fn on the loop and waits for it to finish.
func (c *Core) do(fn func() error) error {
	errCh := make(chan error, 1)
	select {
	case c.cmds <- func() { errCh <- fn() }:
	case <-c.done:
		return ErrStopped
	}
	select {
	case err := <-errCh:
		return err
	case <-c.done:
		return ErrStopped
	}
}

// Inject processes a telephony event synchronously (used by tests and tools).
func (c *Core) Inject(ev domain.TelephonyEvent) {
	_ = c.do(func() error { c.handleEvent(ev); return nil })
}

// Tick runs one timer pass synchronously.
func (c *Core) Tick() { _ = c.do(func() error { c.tick(); return nil }) }

// SetClock overrides the time source (tests only; call before Run).
func (c *Core) SetClock(now func() time.Time) { c.now = now }

func (c *Core) handleEvent(ev domain.TelephonyEvent) {
	switch ev.Type {
	case domain.EventCallStarted:
		c.onStasisStart(ev)
	case domain.EventCallEnded, domain.EventChannelDestroyed:
		c.onChannelGone(ev)
	case domain.EventPlaybackFinished:
		c.onPlaybackFinished(ev.PlaybackID)
	case domain.EventDTMFReceived:
		c.onDTMF(ev.ChannelID, ev.Digit)
	case domain.EventRecordingFinished, domain.EventRecordingFailed:
		c.onRecordingFinished(ev)
	case domain.EventStreamConnected:
		slog.Info("ARI stream (re)connected", "live_calls", len(c.calls))
		c.reconcile()
	}
}

func (c *Core) tick() {
	now := c.now()
	for _, call := range c.calls {
		c.tickCall(call, now)
	}
	c.tickAgents(now)
	c.dispatch()
	c.tickCampaigns(now)
	c.updateGauges()
}
