package ai

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"
)

type toolArgs struct {
	Queue          string `json:"queue"`
	Reason         string `json:"reason"`
	Summary        string `json:"summary"`
	Resolution     string `json:"resolution"`
	Number         string `json:"number"`
	PreferredTime  string `json:"preferred_time"`
	Notes          string `json:"notes"`
	Phone          string `json:"phone"`
	Message        string `json:"message"`
	CallbackNumber string `json:"callback_number"`
	Name           string `json:"name"`
	Topic          string `json:"topic"`
	Request        string `json:"request"`
	Query          string `json:"query"`
	Subject        string `json:"subject"`
	Description    string `json:"description"`
	Priority       string `json:"priority"`
}

// handleTool executes a function call requested by the model.
func (s *Session) handleTool(ctx context.Context, callID, name, rawArgs string) {
	var a toolArgs
	_ = json.Unmarshal([]byte(rawArgs), &a)
	slog.Info("AI tool call", "call_id", s.spec.CallID, "tool", name)
	id := s.spec.CallID
	var result any
	respond := true
	switch name {
	case domain.ToolTransferToQueue:
		s.ending.Store(true)
		respond = false
		result = map[string]any{"ok": true}
		s.afterPlayout(ctx, func() { s.ctl.AITransfer(id, a.Queue, a.Reason, a.Summary) })
	case domain.ToolEndCall:
		s.ending.Store(true)
		respond = false
		result = map[string]any{"ok": true}
		s.afterPlayout(ctx, func() { s.ctl.AIEndCall(id, a.Summary) })
	case domain.ToolCreateCallback:
		notes := a.Notes
		if a.PreferredTime != "" {
			notes += " (preferred time: " + a.PreferredTime + ")"
		}
		s.ctl.AICallback(id, a.Number, strings.TrimSpace(notes))
		result = map[string]any{"ok": true, "message": "Callback scheduled."}
	case domain.ToolTakeMessage:
		num := a.CallbackNumber
		msg := "Message"
		if a.Name != "" {
			msg += " from " + a.Name
		}
		s.ctl.AICallback(id, num, msg+": "+a.Message)
		result = map[string]any{"ok": true, "message": "Message recorded for the team."}
	case domain.ToolLookupContact:
		phone := a.Phone
		if phone == "" {
			phone = s.spec.CallerNumber
		}
		result = s.dir.LookupContact(ctx, phone)
	case domain.ToolRouteCall:
		s.ending.Store(true)
		respond = false
		dec := s.dir.Route(ctx, s.spec, a.Topic, a.Request+"\n"+a.Summary)
		result = map[string]any{"ok": true, "target": dec.TargetName}
		s.afterPlayout(ctx, func() { s.ctl.AIRoute(id, dec.Action, dec.Reason, a.Summary) })
	case domain.ToolSearchKnowledge:
		result = s.dir.SearchKnowledge(ctx, s.spec.Profile, a.Query)
	case domain.ToolCreateTicket:
		result = s.dir.CreateTicket(ctx, s.spec, a.Subject, a.Description, a.Priority, a.Topic)
	case domain.ToolTicketStatus:
		result = s.dir.TicketStatus(ctx, s.spec, a.Number)
	case domain.ToolUpdateTicket:
		result = s.dir.UpdateTicket(ctx, s.spec, a.Number, a.Message)
	case domain.ToolCheckHours:
		result = s.dir.BusinessHours(ctx)
	default:
		result = map[string]any{"error": "unknown tool"}
	}
	out, _ := json.Marshal(result)
	_ = s.send(ctx, map[string]any{"type": "conversation.item.create", "item": map[string]any{
		"type": "function_call_output", "call_id": callID, "output": string(out)}})
	if respond {
		_ = s.send(ctx, map[string]any{"type": "response.create"})
	}
}

// afterPlayout runs fn once the assistant has finished speaking, so the
// caller hears the goodbye or handoff message before the call changes.
func (s *Session) afterPlayout(ctx context.Context, fn func()) {
	go func() {
		deadline := time.Now().Add(20 * time.Second)
		quiet := 0
		for time.Now().Before(deadline) {
			if s.rtp.Pending() == 0 {
				quiet++
				if quiet >= 8 { // ~800 ms of silence: response fully delivered
					break
				}
			} else {
				quiet = 0
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
			}
		}
		fn()
	}()
}
