package ivr

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"callcenter-middleware/internal/config"
	"callcenter-middleware/internal/domain"
)

// Engine routes telephony events and manages call sessions.
type Engine struct {
	cfg      *config.Config
	sw       domain.SwitchController
	notify   domain.ScreenPopNotifier
	metrics  domain.MetricsRecorder
	sessions sync.Map // map[string]*domain.CallSession
	handlers map[domain.CallState]domain.CallStateHandler
}

// NewEngine constructs a new IVR state machine engine.
func NewEngine(cfg *config.Config, sw domain.SwitchController, n domain.ScreenPopNotifier, m domain.MetricsRecorder) *Engine {
	e := &Engine{
		cfg:      cfg,
		sw:       sw,
		notify:   n,
		metrics:  m,
		handlers: make(map[domain.CallState]domain.CallStateHandler),
	}
	e.handlers[domain.StateGreeting] = NewGreetingHandler(e)
	e.handlers[domain.StateRouting] = NewRoutingHandler(e)
	return e
}

// ProcessEvent handles an incoming telephony event.
func (e *Engine) ProcessEvent(ctx context.Context, ev domain.TelephonyEvent) {
	switch ev.Type {
	case domain.EventCallStarted:
		e.handleCallStarted(ctx, ev)
	case domain.EventCallEnded:
		e.handleCallEnded(ev.ChannelID)
	case domain.EventPlaybackFinished:
		e.dispatchPlaybackFinished(ctx, ev)
	case domain.EventDTMFReceived:
		e.dispatchDTMF(ctx, ev)
	}
}

func (e *Engine) handleCallStarted(ctx context.Context, ev domain.TelephonyEvent) {
	sess := &domain.CallSession{
		ChannelID:    ev.ChannelID,
		CallerNumber: ev.CallerNumber,
		CallerName:   ev.CallerName,
		CurrentState: domain.StateGreeting,
		StartTime:    time.Now(),
	}
	e.sessions.Store(ev.ChannelID, sess)
	if e.metrics != nil {
		e.metrics.IncActiveCalls()
	}
	slog.Info("Call session started", "channel_id", sess.ChannelID, "caller_number", sess.CallerNumber)

	if err := e.sw.AnswerChannel(ctx, sess.ChannelID); err != nil {
		slog.Error("Failed to answer channel", "channel_id", sess.ChannelID, "err", err)
	}
	e.transition(ctx, sess, domain.StateGreeting)
}

func (e *Engine) handleCallEnded(channelID string) {
	if val, ok := e.sessions.LoadAndDelete(channelID); ok {
		sess := val.(*domain.CallSession)
		if sess.CancelTimer != nil {
			sess.CancelTimer()
		}
		if e.metrics != nil {
			e.metrics.DecActiveCalls()
		}
		slog.Info("Call session ended", "channel_id", channelID, "duration", time.Since(sess.StartTime))
	}
}

func (e *Engine) transition(ctx context.Context, sess *domain.CallSession, next domain.CallState) {
	sess.CurrentState = next
	if handler, ok := e.handlers[next]; ok {
		if err := handler.OnEnter(ctx, sess); err != nil {
			slog.Error("Handler OnEnter failed", "state", next, "err", err)
		}
	}
}

func (e *Engine) dispatchPlaybackFinished(ctx context.Context, ev domain.TelephonyEvent) {
	e.sessions.Range(func(key, val any) bool {
		sess := val.(*domain.CallSession)
		if sess.ActivePlayID == ev.PlaybackID {
			if handler, ok := e.handlers[sess.CurrentState]; ok {
				if next, err := handler.OnEvent(ctx, sess, ev); err == nil && next != "" && next != sess.CurrentState {
					e.transition(ctx, sess, next)
				}
			}
			return false
		}
		return true
	})
}

func (e *Engine) dispatchDTMF(ctx context.Context, ev domain.TelephonyEvent) {
	if val, ok := e.sessions.Load(ev.ChannelID); ok {
		sess := val.(*domain.CallSession)
		if handler, ok := e.handlers[sess.CurrentState]; ok {
			if next, err := handler.OnEvent(ctx, sess, ev); err == nil && next != "" && next != sess.CurrentState {
				e.transition(ctx, sess, next)
			}
		}
	}
}
