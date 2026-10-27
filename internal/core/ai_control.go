package core

import (
	"callcenter-middleware/internal/domain"
	"context"
	"time"
)

// AITransfer implements AIControl: the AI asked for a human.
func (c *Core) AITransfer(callID, queue, reason, summary string) {
	c.post(func() {
		call := c.calls[callID]
		if call == nil || call.State != domain.CallAI {
			return
		}
		if summary != "" {
			call.Summary = "AI handoff: " + summary
		}
		c.timeline(call, "ai_transfer", queue+" "+reason)
		c.stopAI(call)
		c.aiHandoffQueue(call, queue)
	})
}

// AIEndCall implements AIControl: the AI resolved the call.
func (c *Core) AIEndCall(callID, summary string) {
	c.post(func() {
		call := c.calls[callID]
		if call == nil || call.State != domain.CallAI {
			return
		}
		if summary != "" {
			call.Summary = summary
		}
		call.Outcome = domain.OutcomeAIResolved
		c.finish(call, "ai")
	})
}

// AICallback implements AIControl: the caller wants to be called back.
func (c *Core) AICallback(callID, number, notes string) {
	c.post(func() {
		call := c.calls[callID]
		if call == nil {
			return
		}
		if number == "" {
			number = call.CallerNumber
		}
		cb := &domain.Callback{CallID: callID, Number: number, Name: call.CallerName,
			QueueID: call.aiHandoff, Notes: notes}
		if err := c.Store.CreateCallback(context.Background(), cb); err == nil {
			c.publish(domain.EvCallbackCreated, cb)
			c.timeline(call, "callback_created", number)
		}
	})
}

// AITranscript implements AIControl: persist and broadcast an utterance.
func (c *Core) AITranscript(callID, speaker, text string) {
	t := domain.TranscriptEntry{CallID: callID, Speaker: speaker, Text: text, At: time.Now().UTC()}
	_ = c.Store.AddTranscript(context.Background(), t)
	c.publish(domain.EvTranscript, t)
}

// AIFailed implements AIControl: the realtime session broke.
func (c *Core) AIFailed(callID string, err error) {
	c.post(func() {
		if call := c.calls[callID]; call != nil && call.State == domain.CallAI {
			c.aiFallback(call, err.Error())
		}
	})
}
