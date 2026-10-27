package core

import (
	"time"

	"callcenter-middleware/internal/domain"
)

// startSurvey keeps the caller on the line after the agent hangs up and
// asks for a 1-5 satisfaction score.
func (c *Core) startSurvey(call *liveCall) {
	now := c.now()
	call.talkEnd = now
	if a := c.agents[call.AgentID]; a != nil && a.CallID == call.ID {
		call.TalkSec = secs(now.Sub(*call.AnsweredAt)) - call.HoldSec
		c.agentCallEnded(a, call)
	}
	call.agentChan = ""
	if call.bridgeID != "" {
		c.destroyBridge(call.bridgeID)
		call.bridgeID = ""
	}
	call.State = domain.CallSurvey
	c.timeline(call, "survey", "")
	c.changed(call)
	c.play(call, c.cfg.SurveyMedia, func() {
		call.surveyUntil = c.now().Add(8 * time.Second)
	})
}

func (c *Core) surveyDigit(call *liveCall, digit string) {
	if len(digit) != 1 || digit[0] < '1' || digit[0] > '5' {
		return
	}
	c.stopPlayback(call)
	call.CSAT = int(digit[0] - '0')
	call.surveyUntil = time.Time{}
	c.timeline(call, "csat", digit)
	c.changed(call)
	c.play(call, c.cfg.SurveyThanksMedia, func() {
		c.hangup(call.callerChan, "normal")
		c.finish(call, "survey")
	})
}
