package core

import "log/slog"

// reconcile drops calls whose caller channel no longer exists in Asterisk,
// e.g. after the PBX restarted while the event stream was down.
func (c *Core) reconcile() {
	if len(c.calls) == 0 {
		return
	}
	ctx, cancel := c.opCtx()
	ids, err := c.Switch.ListChannels(ctx)
	cancel()
	if err != nil {
		slog.Warn("Reconcile: list channels failed", "err", err)
		return
	}
	alive := make(map[string]bool, len(ids))
	for _, id := range ids {
		alive[id] = true
	}
	for _, call := range c.calls {
		if call.callerChan != "" && !alive[call.callerChan] {
			slog.Warn("Reconcile: caller channel vanished", "call_id", call.ID)
			c.timeline(call, "reconciled", "channel vanished during ARI outage")
			c.finish(call, "pbx")
		}
	}
}
