package core

import (
	"context"
	"time"

	"callcenter-middleware/internal/domain"
)

// finish ends a call: hangs up remaining legs (unless released to the
// dialplan), frees the agent into wrap-up and writes the final CDR.
func (c *Core) finish(call *liveCall, by string) {
	if call.ended {
		return
	}
	call.ended = true
	c.stopPlayback(call)
	if call.mediaChan != "" && c.AI != nil {
		c.AI.Stop(call.ID)
	}
	released := call.Outcome == domain.OutcomeDialplan
	for chanID, l := range c.legs {
		if l.call != call {
			continue
		}
		delete(c.legs, chanID)
		if l.role == roleCaller && released {
			continue
		}
		c.hangup(chanID, "normal")
		if l.bridgeID != "" {
			c.destroyBridge(l.bridgeID)
		}
	}
	for chanID, uid := range call.ringing {
		delete(call.ringing, chanID)
		if a := c.agents[uid]; a != nil && a.CallID == call.ID {
			a.CallID = ""
			c.setAgentStatus(a, domain.AgentAvailable, "")
		}
	}
	c.destroyBridge(call.bridgeID)

	now := c.now()
	if call.Outcome == "" {
		call.Outcome = outcomeFor(call)
	}
	if !call.holdStart.IsZero() {
		call.HoldSec += secs(now.Sub(call.holdStart))
		call.holdStart = time.Time{}
	}
	talkEnd := now
	if !call.talkEnd.IsZero() {
		talkEnd = call.talkEnd
	}
	if call.AnsweredAt != nil {
		call.TalkSec = secs(talkEnd.Sub(*call.AnsweredAt)) - call.HoldSec
		if call.TalkSec < 0 {
			call.TalkSec = 0
		}
	} else if call.QueuedAt != nil {
		call.WaitSec = secs(now.Sub(*call.QueuedAt))
	}
	call.EndedAt = ptr(now)
	call.HangupBy = by
	call.State = domain.CallEnded

	if a := c.agents[call.AgentID]; a != nil && a.CallID == call.ID {
		c.agentCallEnded(a, call)
	}
	delete(c.calls, call.ID)
	if c.Metrics != nil {
		c.Metrics.DecActiveCalls()
		c.Metrics.RecordOutcome(string(call.Outcome))
	}
	c.timeline(call, "ended", string(call.Outcome)+" by "+by)
	c.persist(call)
	c.publish(domain.EvCallEnded, c.view(call))
	c.updateCallback(call)
	c.updateLead(call)
	if c.Analyzer != nil && (call.AnsweredAt != nil || call.AIProfileID > 0) {
		id := call.ID
		go c.Analyzer.Analyze(context.Background(), id)
	}
	c.dispatch()
}

func outcomeFor(call *liveCall) domain.CallOutcome {
	switch call.State {
	case domain.CallConnected, domain.CallHold:
		return domain.OutcomeAnswered
	case domain.CallQueued, domain.CallRinging:
		return domain.OutcomeAbandoned
	case domain.CallAI:
		return domain.OutcomeAIResolved
	case domain.CallVoicemail:
		return domain.OutcomeVoicemail
	case domain.CallDialing:
		return domain.OutcomeNoAnswer
	case domain.CallIVR:
		return domain.OutcomeHangupIVR
	}
	if call.AnsweredAt != nil {
		return domain.OutcomeAnswered
	}
	return domain.OutcomeAbandoned
}

// updateCallback records the result of a callback attempt.
func (c *Core) updateCallback(call *liveCall) {
	if call.callbackID == 0 {
		return
	}
	ctx, cancel := c.opCtx()
	defer cancel()
	cb, err := c.Store.GetCallback(ctx, call.callbackID)
	if err != nil {
		return
	}
	if call.AnsweredAt != nil {
		cb.Status = domain.CallbackDone
	} else if cb.Attempts >= 3 {
		cb.Status = domain.CallbackFailed
	} else {
		cb.Status = domain.CallbackPending
	}
	_ = c.Store.UpdateCallback(ctx, cb)
	c.publish(domain.EvCallbackCreated, cb)
}
