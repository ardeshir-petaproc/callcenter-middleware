package ivr

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"callcenter-middleware/internal/domain"
)

// GreetingHandler plays the IVR prompt and manages input wait timers.
type GreetingHandler struct {
	engine *Engine
}

// NewGreetingHandler constructs a handler for greeting state.
func NewGreetingHandler(engine *Engine) *GreetingHandler {
	return &GreetingHandler{engine: engine}
}

func (h *GreetingHandler) Name() domain.CallState {
	return domain.StateGreeting
}

func (h *GreetingHandler) OnEnter(ctx context.Context, session *domain.CallSession) error {
	session.ActivePlayID = fmt.Sprintf("play-%s-%d", session.ChannelID, time.Now().UnixNano())
	slog.Info("Playing greeting prompt", "channel_id", session.ChannelID, "playback_id", session.ActivePlayID)
	return h.engine.sw.PlayMedia(ctx, session.ChannelID, h.engine.cfg.GreetingMedia, session.ActivePlayID)
}

func (h *GreetingHandler) OnEvent(ctx context.Context, session *domain.CallSession, ev domain.TelephonyEvent) (domain.CallState, error) {
	switch ev.Type {
	case domain.EventDTMFReceived:
		// Barge-in: immediately cancel playback
		if session.ActivePlayID != "" {
			_ = h.engine.sw.StopPlayback(ctx, session.ActivePlayID)
			session.ActivePlayID = ""
		}
		if session.CancelTimer != nil {
			session.CancelTimer()
		}
		session.InputDigits = ev.Digit
		return domain.StateRouting, nil

	case domain.EventPlaybackFinished:
		session.ActivePlayID = ""
		h.startInputTimer(session)
		return domain.StateGreeting, nil
	}
	return domain.StateGreeting, nil
}

func (h *GreetingHandler) startInputTimer(session *domain.CallSession) {
	timerCtx, cancel := context.WithCancel(context.Background())
	session.CancelTimer = cancel

	go func(chID string, retryCount int) {
		select {
		case <-time.After(h.engine.cfg.IVRTimeout):
			slog.Warn("IVR input timeout elapsed", "channel_id", chID, "retry", retryCount)
			val, ok := h.engine.sessions.Load(chID)
			if !ok {
				return
			}
			sess := val.(*domain.CallSession)
			if sess.CurrentState != domain.StateGreeting {
				return
			}
			if sess.RetryCount < h.engine.cfg.MaxRetries {
				sess.RetryCount++
				h.engine.transition(context.Background(), sess, domain.StateGreeting)
			} else {
				// Fallback to default queue on timeout exhaustion
				if h.engine.metrics != nil {
					h.engine.metrics.RecordDTMF("timeout", "timeout_fallback")
				}
				_ = h.engine.sw.ContinueToQueue(context.Background(), sess.ChannelID, h.engine.cfg.QueueContext, h.engine.cfg.DefaultQueue, 1)
				h.engine.handleCallEnded(sess.ChannelID)
			}
		case <-timerCtx.Done():
			return
		}
	}(session.ChannelID, session.RetryCount)
}
