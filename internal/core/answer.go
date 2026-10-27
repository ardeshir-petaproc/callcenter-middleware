package core

import (
	"log/slog"

	"callcenter-middleware/internal/domain"
)

// onAgentAnswered bridges the caller with the first agent who answers.
func (c *Core) onAgentAnswered(l *leg) {
	call := l.call
	a := c.agents[l.agentID]
	if call.ended || call.agentChan != "" || (call.State != domain.CallQueued && call.State != domain.CallRinging) {
		delete(c.legs, l.chanID)
		delete(call.ringing, l.chanID)
		c.hangup(l.chanID, "normal")
		if a != nil && a.Status == domain.AgentRinging {
			a.CallID = ""
			c.setAgentStatus(a, domain.AgentAvailable, "")
		}
		return
	}
	delete(call.ringing, l.chanID)
	c.cancelRinging(call, l.chanID)
	c.stopPlayback(call)
	c.stopMOH(call)

	if err := c.bridge(call, call.callerChan, l.chanID); err != nil {
		slog.Error("Bridge caller and agent failed", "call_id", call.ID, "err", err)
	}
	now := c.now()
	call.agentChan = l.chanID
	call.AgentID = l.agentID
	call.AnsweredAt = ptr(now)
	call.targetAgent = 0
	if call.QueuedAt != nil {
		call.WaitSec = secs(now.Sub(*call.QueuedAt))
	}
	sla := 20
	if call.queue != nil && call.queue.SLASec > 0 {
		sla = call.queue.SLASec
	}
	call.SLAMet = now.Sub(call.queueEnter).Seconds() <= float64(sla)
	call.State = domain.CallConnected
	if a != nil {
		a.MissedInRow = 0
		a.CallID = call.ID
		c.setAgentStatus(a, domain.AgentBusy, "")
		c.timeline(call, "answered", a.Username)
	}
	if call.queue != nil {
		if c.Metrics != nil {
			c.Metrics.ObserveWait(call.queue.Name, now.Sub(call.queueEnter).Seconds())
			c.Metrics.RecordHandoff(call.queue.Name, "agent_answered")
		}
		if call.queue.Record {
			c.startRecording(call)
		}
	}
	c.changed(call)
}

// bridge creates (or reuses) the call's mixing bridge and adds channels.
func (c *Core) bridge(call *liveCall, chans ...string) error {
	ctx, cancel := c.opCtx()
	defer cancel()
	if call.bridgeID == "" {
		id := "br-" + call.ID
		if err := c.Switch.CreateBridge(ctx, id, "mixing"); err != nil {
			return err
		}
		call.bridgeID = id
	}
	return c.Switch.AddToBridge(ctx, call.bridgeID, chans...)
}

func (c *Core) startRecording(call *liveCall) {
	if call.bridgeID == "" || call.Recording != "" {
		return
	}
	name := "call-" + call.ID
	ctx, cancel := c.opCtx()
	defer cancel()
	if err := c.Switch.RecordBridge(ctx, call.bridgeID, name); err != nil {
		slog.Warn("Start recording failed", "call_id", call.ID, "err", err)
		return
	}
	call.Recording = name
	c.timeline(call, "recording", name)
}

func (c *Core) onChannelGone(ev domain.TelephonyEvent) {
	if l, ok := c.legs[ev.ChannelID]; ok {
		c.onLegGone(l, ev.Cause)
	}
}

// onLegGone reacts to a channel of a call hanging up or failing to answer.
func (c *Core) onLegGone(l *leg, cause int) {
	if _, ok := c.legs[l.chanID]; !ok {
		return
	}
	delete(c.legs, l.chanID)
	call := l.call
	switch l.role {
	case roleCaller:
		c.finish(call, "caller")
	case roleAgent:
		if !l.answered {
			delete(call.ringing, l.chanID)
			c.agentMissed(call, l.agentID, cause)
			if call.State == domain.CallRinging && len(call.ringing) == 0 {
				call.State = domain.CallQueued
				c.changed(call)
			}
			c.dispatch()
			return
		}
		if l.chanID == call.agentChan && !call.transferring {
			c.finish(call, "agent")
		}
	case roleOutAgent:
		if !l.answered {
			call.Outcome = domain.OutcomeFailed
		}
		c.finish(call, "agent")
	case roleOutCust:
		if !l.answered {
			call.Outcome = domain.OutcomeNoAnswer
		}
		c.finish(call, "customer")
	case roleExternal:
		c.onExternalGone(call, l)
	case roleMedia:
		if call.State == domain.CallAI && !call.ended {
			c.aiFallback(call, "media channel lost")
		}
	case roleSupervisor, roleSnoop:
		c.endMonitor(l)
	}
}

func (c *Core) agentMissed(call *liveCall, userID int64, cause int) {
	a := c.agents[userID]
	if a == nil || a.CallID != call.ID || a.Status != domain.AgentRinging {
		return
	}
	a.CallID = ""
	a.MissedInRow++
	c.timeline(call, "agent_no_answer", a.Username)
	if c.cfg.AutoPauseMisses > 0 && a.MissedInRow >= c.cfg.AutoPauseMisses {
		c.setAgentStatus(a, domain.AgentPaused, "Missed calls")
		return
	}
	c.setAgentStatus(a, domain.AgentAvailable, "")
}
