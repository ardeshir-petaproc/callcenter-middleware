package core

import (
	"context"
	"log/slog"
	"strconv"

	"callcenter-middleware/internal/domain"
)

func itoa(v int64) string { return strconv.FormatInt(v, 10) }

func atoi(s string) int64 {
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

// execute performs a routing action for a caller, optionally after media.
func (c *Core) execute(call *liveCall, a domain.Action) {
	if a.Media != "" && a.Type != domain.ActionHangup {
		media := a.Media
		a.Media = ""
		c.play(call, media, func() { c.execute(call, a) })
		return
	}
	switch a.Type {
	case domain.ActionQueue:
		q := c.cache.findQueue(a.Target)
		if q == nil || !q.Enabled {
			slog.Warn("Queue not found or disabled, using default", "target", a.Target)
			def := c.defaultAction()
			if def.Type == domain.ActionQueue && def.Target != a.Target {
				c.execute(call, def)
				return
			}
			c.toDialplan(call, c.cfg.DefaultQueue, "queue_missing")
			return
		}
		if q.Mode == domain.QueueModeAsterisk {
			call.QueueID = q.ID
			c.toDialplan(call, q.Extension, "dtmf_selection")
			return
		}
		c.enqueue(call, q)
	case domain.ActionDialplan:
		c.toDialplan(call, a.Target, "dtmf_selection")
	case domain.ActionAI:
		c.startAI(call, atoi(a.Target))
	case domain.ActionVoicemail:
		c.startVoicemail(call)
	case domain.ActionCallback:
		c.requestCallback(call, "Requested from IVR")
	case domain.ActionHangup:
		c.play(call, a.Media, func() { c.hangup(call.callerChan, "normal") })
	case domain.ActionFlow:
		if f := c.cache.flowByID(atoi(a.Target)); f != nil {
			call.flow, call.FlowID, call.retries = f, f.ID, 0
			c.timeline(call, "flow", f.Name)
			c.playGreeting(call)
			return
		}
		c.execute(call, c.defaultAction())
	case domain.ActionAgent:
		c.directToAgent(call, atoi(a.Target))
	case domain.ActionExternal:
		c.transferExternal(call, a.Target)
	default:
		c.execute(call, c.defaultAction())
	}
}

// toDialplan releases the caller to an Asterisk dialplan extension (the
// original Phase 1 behaviour and the fail-open path).
func (c *Core) toDialplan(call *liveCall, ext, reason string) {
	c.stopPlayback(call)
	if c.Metrics != nil {
		c.Metrics.RecordHandoff(ext, reason)
	}
	c.screenPop(call, ext, reason)
	c.timeline(call, "dialplan", ext)
	ctx, cancel := c.opCtx()
	err := c.Switch.ContinueToQueue(ctx, call.callerChan, c.cfg.QueueContext, ext, 1)
	cancel()
	if err != nil {
		slog.Error("Continue to dialplan failed", "call_id", call.ID, "ext", ext, "err", err)
	}
	call.Outcome = domain.OutcomeDialplan
	c.finish(call, "middleware")
}

// screenPop sends the legacy handoff webhook for external screen-pop tools.
func (c *Core) screenPop(call *liveCall, target, reason string) {
	if c.Notifier == nil {
		return
	}
	n := domain.HandoffNotification{
		ChannelID: call.callerChan, CallID: call.ID, CallerNumber: call.CallerNumber,
		CallerName: call.CallerName, TargetQueue: target, InputDigits: call.IVRPath,
		Reason: reason, Timestamp: c.now().UTC(),
	}
	go func() { _ = c.Notifier.NotifyQueueHandoff(context.Background(), n) }()
}

// requestCallback stores a callback for the caller and ends the call.
func (c *Core) requestCallback(call *liveCall, notes string) {
	cb := &domain.Callback{CallID: call.ID, Number: call.CallerNumber, Name: call.CallerName,
		QueueID: call.QueueID, Notes: notes}
	ctx, cancel := c.opCtx()
	err := c.Store.CreateCallback(ctx, cb)
	cancel()
	if err != nil {
		slog.Error("Create callback failed", "call_id", call.ID, "err", err)
	} else {
		c.publish(domain.EvCallbackCreated, cb)
	}
	call.Outcome = domain.OutcomeCallback
	call.State = domain.CallMessage
	c.changed(call)
	c.play(call, c.cfg.CallbackMedia, func() { c.hangup(call.callerChan, "normal") })
}

// Itoa formats an id as a string (exported for tests and API helpers).
func Itoa(v int64) string { return itoa(v) }
