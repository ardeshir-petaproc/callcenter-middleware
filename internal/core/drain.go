package core

import (
	"context"
	"log/slog"

	"callcenter-middleware/internal/domain"
)

// DrainAndRelease fails open on shutdown: every caller that is not yet
// talking to a human (IVR, AI, queue, voicemail prompt) is released to the
// default Asterisk queue so nobody is dropped. Bridged conversations are
// left intact in Asterisk and continue without the middleware.
func (c *Core) DrainAndRelease(ctx context.Context) {
	slog.Info("Starting zero-drop graceful call drain")
	done := make(chan int, 1)
	err := c.do(func() error {
		n := 0
		for _, call := range c.calls {
			switch call.State {
			case domain.CallIVR, domain.CallAI, domain.CallQueued, domain.CallRinging,
				domain.CallVoicemail, domain.CallMessage:
			default:
				continue
			}
			n++
			if call.State == domain.CallAI {
				c.stopAI(call)
			}
			c.leaveQueue(call)
			slog.Info("Draining active call to default queue", "call_id", call.ID, "queue", c.cfg.DefaultQueue)
			if c.Metrics != nil {
				c.Metrics.RecordHandoff(c.cfg.DefaultQueue, "shutdown_drain")
			}
			ctx2, cancel := context.WithTimeout(ctx, c.cfg.DrainTimeout)
			_ = c.Switch.ContinueToQueue(ctx2, call.callerChan, c.cfg.QueueContext, c.cfg.DefaultQueue, 1)
			cancel()
			call.Outcome = domain.OutcomeDialplan
			c.finish(call, "shutdown")
		}
		done <- n
		return nil
	})
	if err != nil {
		slog.Warn("Call drain skipped", "err", err)
		return
	}
	select {
	case n := <-done:
		slog.Info("All active calls drained successfully", "drained", n)
	case <-ctx.Done():
		slog.Warn("Call drain timed out", "err", ctx.Err())
	}
}
