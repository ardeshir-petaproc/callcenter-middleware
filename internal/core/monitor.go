package core

import (
	"errors"
	"log/slog"

	"callcenter-middleware/internal/domain"
)

// Monitor rings a supervisor and, on answer, lets them listen (spy), coach
// the agent without the caller hearing (whisper) or join the call (barge).
func (c *Core) Monitor(callID string, supervisorID int64, mode string) error {
	if mode != MonitorSpy && mode != MonitorWhisper && mode != MonitorBarge {
		return errors.New("mode must be spy, whisper or barge")
	}
	return c.withCall(callID, func(call *liveCall) error {
		if call.State != domain.CallConnected && call.State != domain.CallHold {
			return ErrInvalidState
		}
		sup := c.agents[supervisorID]
		if sup == nil || sup.TenantID != call.TenantID {
			return ErrCallNotFound
		}
		if sup.user.DialEndpoint() == "" {
			return errors.New("supervisor has no extension")
		}
		chanID := c.nextChan("sup", call.ID)
		c.legs[chanID] = &leg{chanID: chanID, call: call, role: roleSupervisor, agentID: supervisorID,
			mode: mode, since: c.now()}
		ctx, cancel := c.opCtx()
		defer cancel()
		err := c.Switch.Originate(ctx, domain.OriginateRequest{
			Endpoint: sup.user.DialEndpoint(), ChannelID: chanID, TimeoutSec: 30,
			CallerID: "\"" + mode + "\" <" + call.CallerNumber + ">", AppArgs: roleSupervisor + "," + call.ID,
		})
		if err != nil {
			delete(c.legs, chanID)
			return err
		}
		c.timeline(call, "monitor_"+mode, sup.Username)
		return nil
	})
}

func (c *Core) onSupervisorAnswered(l *leg) {
	call := l.call
	ctx, cancel := c.opCtx()
	defer cancel()
	if l.mode == MonitorBarge {
		if err := c.Switch.AddToBridge(ctx, call.bridgeID, l.chanID); err != nil {
			slog.Warn("Barge failed", "call_id", call.ID, "err", err)
			c.hangup(l.chanID, "normal")
		}
		c.changed(call)
		return
	}
	target := call.agentChan
	if target == "" {
		target = call.callerChan
	}
	snoopID := c.nextChan("snoop", call.ID)
	whisper := "none"
	if l.mode == MonitorWhisper {
		whisper = "out"
	}
	c.legs[snoopID] = &leg{chanID: snoopID, call: call, role: roleSnoop, pairChan: l.chanID, since: c.now()}
	l.pairChan = snoopID
	err := c.Switch.Snoop(ctx, domain.SnoopRequest{
		ChannelID: target, SnoopID: snoopID, Spy: "both", Whisper: whisper, AppArgs: roleSnoop + "," + call.ID,
	})
	if err != nil {
		slog.Warn("Snoop failed", "call_id", call.ID, "err", err)
		delete(c.legs, snoopID)
		c.hangup(l.chanID, "normal")
	}
	c.changed(call)
}

func (c *Core) onSnoopReady(l *leg) {
	sup := c.legs[l.pairChan]
	if sup == nil {
		c.hangup(l.chanID, "normal")
		return
	}
	bridgeID := "mon-" + l.chanID
	ctx, cancel := c.opCtx()
	defer cancel()
	if err := c.Switch.CreateBridge(ctx, bridgeID, "mixing"); err != nil {
		slog.Warn("Monitor bridge failed", "err", err)
		return
	}
	sup.bridgeID = bridgeID
	_ = c.Switch.AddToBridge(ctx, bridgeID, sup.chanID, l.chanID)
}

// endMonitor tears down the supervisor/snoop pair when either side ends.
func (c *Core) endMonitor(l *leg) {
	if l.bridgeID != "" {
		c.destroyBridge(l.bridgeID)
	}
	if pair := c.legs[l.pairChan]; pair != nil {
		delete(c.legs, pair.chanID)
		if pair.bridgeID != "" {
			c.destroyBridge(pair.bridgeID)
		}
		c.hangup(pair.chanID, "normal")
	}
	if !l.call.ended {
		c.changed(l.call)
	}
}
