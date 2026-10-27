package ivr

import (
	"context"
	"log/slog"
	"time"

	"callcenter-middleware/internal/domain"
)

// RoutingHandler routes call according to DTMF input.
type RoutingHandler struct {
	engine *Engine
}

// NewRoutingHandler constructs a new routing state handler.
func NewRoutingHandler(engine *Engine) *RoutingHandler {
	return &RoutingHandler{engine: engine}
}

func (h *RoutingHandler) Name() domain.CallState {
	return domain.StateRouting
}

func (h *RoutingHandler) OnEnter(ctx context.Context, session *domain.CallSession) error {
	digit := session.InputDigits
	queue, exists := h.engine.cfg.QueueMappings[digit]
	outcome := "routed"
	reason := "dtmf_selection"

	if !exists {
		if session.RetryCount < h.engine.cfg.MaxRetries {
			session.RetryCount++
			if h.engine.metrics != nil {
				h.engine.metrics.RecordDTMF(digit, "invalid")
			}
			slog.Warn("Invalid digit received, repeating greeting", "channel_id", session.ChannelID, "digit", digit)
			h.engine.transition(ctx, session, domain.StateGreeting)
			return nil
		}
		// Max retries exceeded, route to default queue
		queue = h.engine.cfg.DefaultQueue
		outcome = "invalid_fallback"
		reason = "max_retries_fallback"
	}

	if h.engine.metrics != nil {
		h.engine.metrics.RecordDTMF(digit, outcome)
		h.engine.metrics.RecordHandoff(queue, reason)
	}

	// Trigger asynchronous screen-pop notification
	if h.engine.notify != nil {
		go func() {
			notif := domain.HandoffNotification{
				ChannelID:    session.ChannelID,
				CallerNumber: session.CallerNumber,
				CallerName:   session.CallerName,
				TargetQueue:  queue,
				InputDigits:  digit,
				Reason:       reason,
				Timestamp:    time.Now(),
			}
			_ = h.engine.notify.NotifyQueueHandoff(context.Background(), notif)
		}()
	}

	slog.Info("Transferring call to queue", "channel_id", session.ChannelID, "queue", queue, "reason", reason)
	err := h.engine.sw.ContinueToQueue(ctx, session.ChannelID, h.engine.cfg.QueueContext, queue, 1)
	h.engine.handleCallEnded(session.ChannelID)
	return err
}

func (h *RoutingHandler) OnEvent(ctx context.Context, session *domain.CallSession, ev domain.TelephonyEvent) (domain.CallState, error) {
	return domain.StateRouting, nil
}
