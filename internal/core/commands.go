package core

import (
	"errors"
	"time"

	"callcenter-middleware/internal/domain"
)

// Command errors returned to the API.
var (
	ErrCallNotFound = errors.New("call not found")
	ErrInvalidState = errors.New("operation not allowed in current call state")
)

func (c *Core) withCall(callID string, fn func(*liveCall) error) error {
	return c.do(func() error {
		call := c.calls[callID]
		if call == nil {
			return ErrCallNotFound
		}
		return fn(call)
	})
}

// Hangup terminates a live call.
func (c *Core) Hangup(callID string) error {
	return c.withCall(callID, func(call *liveCall) error {
		c.timeline(call, "hangup_requested", "api")
		c.finish(call, "supervisor")
		return nil
	})
}

// Transfer moves a call to a queue, agent, external number, AI agent or voicemail.
func (c *Core) Transfer(callID string, target domain.Action) error {
	return c.withCall(callID, func(call *liveCall) error {
		switch target.Type {
		case domain.ActionQueue, domain.ActionAgent, domain.ActionExternal, domain.ActionAI,
			domain.ActionVoicemail, domain.ActionDialplan, domain.ActionHangup:
		default:
			return errors.New("unsupported transfer target")
		}
		if call.Direction == domain.DirectionOutbound && target.Type != domain.ActionHangup &&
			target.Type != domain.ActionExternal && target.Type != domain.ActionQueue && target.Type != domain.ActionAgent {
			return ErrInvalidState
		}
		c.timeline(call, "transfer", string(target.Type)+":"+target.Target)
		prev := call.AgentID
		c.detachForTransfer(call)
		call.returnTo = 0
		if target.Type == domain.ActionAgent && prev != 0 && target.Target != itoa(prev) {
			call.returnTo = prev
		}
		c.execute(call, target)
		return nil
	})
}

// detachForTransfer releases the agent/AI side of a call so the caller can
// be routed somewhere else.
func (c *Core) detachForTransfer(call *liveCall) {
	switch call.State {
	case domain.CallAI:
		c.stopAI(call)
	case domain.CallQueued, domain.CallRinging:
		c.leaveQueue(call)
	case domain.CallIVR:
		c.stopPlayback(call)
	case domain.CallHold:
		c.stopMOH(call)
		c.endHold(call)
	}
	if call.agentChan != "" {
		agentChan := call.agentChan
		delete(c.legs, agentChan)
		call.agentChan = ""
		call.transferring = true
		c.hangup(agentChan, "normal")
		call.transferring = false
		if a := c.agents[call.AgentID]; a != nil && a.CallID == call.ID {
			c.agentCallEnded(a, call)
		}
		call.AnsweredAt = nil
		call.AgentID = 0
	}
	if call.bridgeID != "" {
		c.destroyBridge(call.bridgeID)
		call.bridgeID = ""
	}
	call.Recording = ""
}

// Hold places the caller on music on hold, or resumes the conversation.
func (c *Core) Hold(callID string, on bool) error {
	return c.withCall(callID, func(call *liveCall) error {
		ctx, cancel := c.opCtx()
		defer cancel()
		if on {
			if call.State != domain.CallConnected || call.bridgeID == "" {
				return ErrInvalidState
			}
			if err := c.Switch.RemoveFromBridge(ctx, call.bridgeID, call.callerChan); err != nil {
				return err
			}
			call.holdStart = c.now()
			call.State = domain.CallHold
			_ = c.Switch.StartMOH(ctx, call.callerChan, "")
			c.timeline(call, "hold", "")
		} else {
			if call.State != domain.CallHold {
				return ErrInvalidState
			}
			_ = c.Switch.StopMOH(ctx, call.callerChan)
			if err := c.Switch.AddToBridge(ctx, call.bridgeID, call.callerChan); err != nil {
				return err
			}
			c.endHold(call)
			call.State = domain.CallConnected
			c.timeline(call, "unhold", "")
		}
		c.changed(call)
		return nil
	})
}

func (c *Core) endHold(call *liveCall) {
	if !call.holdStart.IsZero() {
		call.HoldSec += secs(c.now().Sub(call.holdStart))
		call.holdStart = time.Time{}
	}
}

// Record starts recording a connected call on demand.
func (c *Core) Record(callID string) error {
	return c.withCall(callID, func(call *liveCall) error {
		if call.bridgeID == "" {
			return ErrInvalidState
		}
		c.startRecording(call)
		c.changed(call)
		return nil
	})
}

// LiveCall returns the live view of a call, if it is active.
func (c *Core) LiveCall(callID string) (CallView, bool) {
	var v CallView
	found := false
	_ = c.do(func() error {
		if call := c.calls[callID]; call != nil {
			v, found = c.view(call), true
		}
		return nil
	})
	return v, found
}
