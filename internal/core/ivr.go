package core

import (
	"time"

	"callcenter-middleware/internal/domain"
)

func (c *Core) playGreeting(call *liveCall) {
	call.State = domain.CallIVR
	call.inputDeadline = time.Time{}
	c.play(call, call.flow.GreetingMedia, func() {
		timeout := call.flow.TimeoutSec
		if timeout <= 0 {
			timeout = 5
		}
		call.inputDeadline = c.now().Add(time.Duration(timeout) * time.Second)
	})
	if call.flow.GreetingMedia == "" && call.inputDeadline.IsZero() {
		call.inputDeadline = c.now().Add(5 * time.Second)
	}
}

func (c *Core) onDTMF(chanID, digit string) {
	l, ok := c.legs[chanID]
	if !ok || l.role != roleCaller {
		return
	}
	call := l.call
	switch call.State {
	case domain.CallIVR:
		c.ivrDigit(call, digit)
	case domain.CallSurvey:
		c.surveyDigit(call, digit)
	case domain.CallQueued, domain.CallRinging:
		if digit == "*" && call.queue != nil && call.queue.AllowCallback {
			c.leaveQueue(call)
			c.timeline(call, "callback_from_queue", "caller pressed *")
			c.requestCallback(call, "Requested while waiting in queue")
		}
	}
}

func (c *Core) ivrDigit(call *liveCall, digit string) {
	// Barge-in: stop the prompt as soon as the caller presses a key.
	c.stopPlayback(call)
	call.inputDeadline = time.Time{}
	call.IVRPath += digit
	action, ok := call.flow.Options[digit]
	if ok {
		if c.Metrics != nil {
			c.Metrics.RecordDTMF(digit, "routed")
		}
		c.timeline(call, "dtmf", digit)
		c.execute(call, action)
		return
	}
	if c.Metrics != nil {
		c.Metrics.RecordDTMF(digit, "invalid")
	}
	c.timeline(call, "dtmf_invalid", digit)
	c.retryOrFallback(call, call.flow.InvalidMedia)
}

func (c *Core) retryOrFallback(call *liveCall, media string) {
	if call.retries < call.flow.MaxRetries {
		call.retries++
		if media == "" {
			c.playGreeting(call)
		} else {
			c.play(call, media, func() { c.playGreeting(call) })
		}
		return
	}
	action := call.flow.TimeoutAction
	if action.Type == domain.ActionNone {
		action = c.defaultAction(call.TenantID)
	}
	if c.Metrics != nil {
		c.Metrics.RecordDTMF("timeout", "timeout_fallback")
	}
	c.timeline(call, "ivr_fallback", string(action.Type)+":"+action.Target)
	c.execute(call, action)
}

// defaultAction is used when configuration leaves a decision unset.
func (c *Core) defaultAction(tenant int64) domain.Action {
	for _, q := range c.cache.sortedQueues(tenant) {
		if q.Enabled {
			return domain.Action{Type: domain.ActionQueue, Target: itoa(q.ID)}
		}
	}
	return domain.Action{Type: domain.ActionDialplan, Target: c.cfg.DefaultQueue}
}

func (c *Core) tickCall(call *liveCall, now time.Time) {
	switch call.State {
	case domain.CallIVR:
		if !call.inputDeadline.IsZero() && now.After(call.inputDeadline) {
			call.inputDeadline = time.Time{}
			c.timeline(call, "ivr_timeout", "")
			c.retryOrFallback(call, "")
		}
	case domain.CallQueued, domain.CallRinging:
		c.tickQueued(call, now)
	case domain.CallAI:
		c.tickAI(call, now)
	case domain.CallSurvey:
		if !call.surveyUntil.IsZero() && now.After(call.surveyUntil) {
			c.hangup(call.callerChan, "normal")
			c.finish(call, "survey_timeout")
		}
	}
	for chanID, l := range c.legs {
		if l.call != call || l.answered || l.role == roleCaller {
			continue
		}
		// Safety net for originations whose destroy event never arrives.
		if now.Sub(l.since) > 90*time.Second {
			c.hangup(chanID, "no_answer")
			c.onLegGone(l, 19)
		}
	}
}
