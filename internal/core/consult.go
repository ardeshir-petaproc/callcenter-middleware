package core

import (
	"errors"
	"log/slog"

	"callcenter-middleware/internal/domain"
)

// Consult starts a warm transfer: the caller is held while the agent talks
// to another agent or an outside number, then completes the transfer,
// joins everyone in a conference, or cancels and returns to the caller.
func (c *Core) Consult(callID string, target domain.Action) error {
	return c.withCall(callID, func(call *liveCall) error {
		if call.State != domain.CallConnected || call.agentChan == "" || call.consultChan != "" {
			return ErrInvalidState
		}
		chanID := c.nextChan(roleConsult, call.ID)
		l := &leg{chanID: chanID, call: call, role: roleConsult, since: c.now()}
		endpoint := ""
		switch target.Type {
		case domain.ActionAgent:
			a := c.agents[int64(atoi(target.Target))]
			if a == nil || a.TenantID != call.TenantID || a.UserID == call.AgentID || !c.ringable(a) {
				return errors.New("agent is not available")
			}
			endpoint, l.agentID = a.user.DialEndpoint(), a.UserID
			a.CallID = call.ID
			c.setAgentStatus(a, domain.AgentRinging, "")
		case domain.ActionExternal:
			if NormalizePhone(target.Target) == "" {
				return errors.New("invalid number")
			}
			endpoint = c.dialString(target.Target)
		default:
			return errors.New("consult target must be an agent or a number")
		}
		c.holdCaller(call)
		c.legs[chanID], call.consultChan = l, chanID
		ctx, cancel := c.opCtx()
		err := c.Switch.Originate(ctx, domain.OriginateRequest{Endpoint: endpoint, ChannelID: chanID,
			CallerID: c.cfg.OutboundCallerID, TimeoutSec: 30, AppArgs: roleConsult + "," + call.ID})
		cancel()
		if err != nil {
			c.dropConsult(call, l)
			c.resumeCaller(call)
			return err
		}
		c.timeline(call, "consult", string(target.Type)+":"+target.Target)
		c.changed(call)
		return nil
	})
}

func (c *Core) onConsultAnswered(l *leg) {
	call := l.call
	if call.ended || call.consultChan != l.chanID {
		delete(c.legs, l.chanID)
		c.hangup(l.chanID, "normal")
		return
	}
	if err := c.bridge(call, l.chanID); err != nil {
		slog.Error("Bridge consult leg failed", "call_id", call.ID, "err", err)
	}
	if a := c.agents[l.agentID]; a != nil {
		c.setAgentStatus(a, domain.AgentBusy, "")
	}
	c.timeline(call, "consult_answered", "")
	c.changed(call)
}

// onConsultGone handles the consulted party hanging up or not answering:
// the caller goes back to the original agent.
func (c *Core) onConsultGone(call *liveCall, l *leg) {
	if call.consultChan != l.chanID {
		return
	}
	c.releaseConsultAgent(call, l)
	call.consultChan = ""
	if call.conference {
		call.conference = false
		c.timeline(call, "conference_left", "")
	} else {
		c.timeline(call, map[bool]string{true: "consult_ended", false: "consult_failed"}[l.answered], "")
		c.resumeCaller(call)
	}
	c.changed(call)
}

// CancelConsult hangs up the consulted party and returns to the caller.
func (c *Core) CancelConsult(callID string) error {
	return c.withCall(callID, func(call *liveCall) error {
		l := c.legs[call.consultChan]
		if l == nil {
			return ErrInvalidState
		}
		c.dropConsult(call, l)
		call.conference = false
		c.timeline(call, "consult_cancelled", "")
		c.resumeCaller(call)
		c.changed(call)
		return nil
	})
}

// CompleteTransfer hands the caller to the consulted party and releases
// the original agent.
func (c *Core) CompleteTransfer(callID string) error {
	return c.withCall(callID, func(call *liveCall) error {
		l := c.legs[call.consultChan]
		if l == nil || !l.answered {
			return ErrInvalidState
		}
		c.releaseAgent(call)
		c.promoteConsult(call, l)
		c.timeline(call, "transfer_completed", "")
		c.changed(call)
		return nil
	})
}

// Conference joins the caller, the agent and the consulted party.
func (c *Core) Conference(callID string) error {
	return c.withCall(callID, func(call *liveCall) error {
		l := c.legs[call.consultChan]
		if l == nil || !l.answered || call.conference {
			return ErrInvalidState
		}
		call.conference = true
		c.resumeCaller(call)
		c.timeline(call, "conference", "")
		c.changed(call)
		return nil
	})
}
