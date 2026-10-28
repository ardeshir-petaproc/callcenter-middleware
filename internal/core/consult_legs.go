package core

import "callcenter-middleware/internal/domain"

// holdCaller takes the caller out of the conversation onto music on hold.
func (c *Core) holdCaller(call *liveCall) {
	ctx, cancel := c.opCtx()
	defer cancel()
	_ = c.Switch.RemoveFromBridge(ctx, call.bridgeID, call.callerChan)
	_ = c.Switch.StartMOH(ctx, call.callerChan, "")
	call.holdStart = c.now()
	call.State = domain.CallHold
}

// resumeCaller brings a held caller back into the call's bridge.
func (c *Core) resumeCaller(call *liveCall) {
	if call.State != domain.CallHold {
		return
	}
	ctx, cancel := c.opCtx()
	defer cancel()
	_ = c.Switch.StopMOH(ctx, call.callerChan)
	_ = c.Switch.AddToBridge(ctx, call.bridgeID, call.callerChan)
	c.endHold(call)
	call.State = domain.CallConnected
}

// dropConsult hangs up the consulted party.
func (c *Core) dropConsult(call *liveCall, l *leg) {
	delete(c.legs, l.chanID)
	c.hangup(l.chanID, "normal")
	c.releaseConsultAgent(call, l)
	call.consultChan = ""
}

func (c *Core) releaseConsultAgent(call *liveCall, l *leg) {
	a := c.agents[l.agentID]
	if a == nil || a.CallID != call.ID || l.agentID == call.AgentID {
		return
	}
	if !l.answered {
		c.agentMissed(call, l.agentID, 0)
		return
	}
	a.CallID = ""
	c.setAgentStatus(a, domain.AgentAvailable, "")
}

// releaseAgent disconnects the call's current agent, who goes to wrap-up.
func (c *Core) releaseAgent(call *liveCall) {
	if call.agentChan != "" {
		delete(c.legs, call.agentChan)
		c.hangup(call.agentChan, "normal")
		call.agentChan = ""
	}
	if a := c.agents[call.AgentID]; a != nil && a.CallID == call.ID {
		c.agentCallEnded(a, call)
	}
	call.AgentID = 0
}

// promoteConsult makes the consulted party the call's agent (or external
// party) and reconnects the caller.
func (c *Core) promoteConsult(call *liveCall, l *leg) {
	call.consultChan, call.conference = "", false
	if l.agentID != 0 {
		l.role, call.agentChan, call.AgentID = roleAgent, l.chanID, l.agentID
	} else {
		l.role, call.extChan = roleExternal, l.chanID
	}
	c.resumeCaller(call)
}

// agentLeftConsult handles the original agent hanging up mid-consultation:
// an answered consult takes over the caller (an attended transfer by
// hanging up); otherwise the caller goes back to the queue.
func (c *Core) agentLeftConsult(call *liveCall) {
	call.agentChan = ""
	if a := c.agents[call.AgentID]; a != nil && a.CallID == call.ID {
		c.agentCallEnded(a, call)
	}
	call.AgentID = 0
	l := c.legs[call.consultChan]
	if l != nil && l.answered {
		c.promoteConsult(call, l)
		c.timeline(call, "transfer_completed", "agent hung up")
		c.changed(call)
		return
	}
	if l != nil {
		c.dropConsult(call, l)
	}
	c.stopMOH(call)
	c.endHold(call)
	c.timeline(call, "consult_abandoned", "")
	if call.queue != nil {
		c.enqueue(call, call.queue)
		return
	}
	c.execute(call, c.defaultAction(call.TenantID))
}

// returnToPrevious redirects a failed blind transfer to an agent back to
// the agent who transferred the call, if they can take it.
func (c *Core) returnToPrevious(call *liveCall) bool {
	prev := c.agents[call.returnTo]
	if call.returnTo == 0 || prev == nil {
		return false
	}
	call.returnTo = 0
	if prev.Status == domain.AgentWrapup && prev.CallID == "" {
		c.setAgentStatus(prev, domain.AgentAvailable, "")
	}
	if !c.ringable(prev) {
		return false
	}
	c.timeline(call, "transfer_failed_return", prev.Username)
	call.targetAgent = prev.UserID
	delete(call.tried, prev.UserID)
	return true
}
