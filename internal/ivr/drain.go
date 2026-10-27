package ivr

import (
	"context"
	"log/slog"
	"sync"

	"callcenter-middleware/internal/domain"
)

// DrainAndRelease safely transfers all active IVR calls to default queue on shutdown.
func (e *Engine) DrainAndRelease(ctx context.Context) {
	slog.Info("Starting zero-drop graceful call drain")
	var wg sync.WaitGroup

	e.sessions.Range(func(key, val any) bool {
		sess := val.(*domain.CallSession)
		wg.Add(1)

		go func(s *domain.CallSession) {
			defer wg.Done()
			if s.CancelTimer != nil {
				s.CancelTimer()
			}
			if s.ActivePlayID != "" {
				_ = e.sw.StopPlayback(ctx, s.ActivePlayID)
			}

			slog.Info("Draining active call to default queue", "channel_id", s.ChannelID, "queue", e.cfg.DefaultQueue)
			if e.metrics != nil {
				e.metrics.RecordHandoff(e.cfg.DefaultQueue, "shutdown_drain")
			}
			_ = e.sw.ContinueToQueue(ctx, s.ChannelID, e.cfg.QueueContext, e.cfg.DefaultQueue, 1)
			e.handleCallEnded(s.ChannelID)
		}(sess)

		return true
	})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		slog.Info("All active calls drained successfully")
	case <-ctx.Done():
		slog.Warn("Call drain timed out", "err", ctx.Err())
	}
}
