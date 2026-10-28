package ai

import (
	"fmt"
	"strings"

	"callcenter-middleware/internal/config"
	"callcenter-middleware/internal/domain"
)

// SessionContext carries per-call facts injected into the instructions.
type SessionContext struct {
	CallerNumber string
	CallerName   string
	Queues       []string
	Topics       []string
	Phone        bool
}

// BuildSession returns a GA Realtime API session object for a profile.
func BuildSession(cfg *config.Config, p domain.AIProfile, sc SessionContext) map[string]any {
	model := p.Model
	if model == "" {
		model = cfg.OpenAIRealtimeModel
	}
	voice := p.Voice
	if voice == "" {
		voice = cfg.OpenAIVoice
	}
	input := map[string]any{
		"turn_detection": map[string]any{
			"type": "server_vad", "silence_duration_ms": 500, "prefix_padding_ms": 300,
			"create_response": true, "interrupt_response": true,
		},
		"transcription": map[string]any{"model": cfg.OpenAITranscribeModel},
	}
	output := map[string]any{"voice": voice}
	if sc.Phone {
		input["format"] = map[string]any{"type": "audio/pcmu"}
		input["noise_reduction"] = map[string]any{"type": "near_field"}
		output["format"] = map[string]any{"type": "audio/pcmu"}
	}
	if p.Language != "" {
		input["transcription"].(map[string]any)["language"] = p.Language
	}
	session := map[string]any{
		"type":              "realtime",
		"model":             model,
		"instructions":      Instructions(p, sc),
		"output_modalities": []string{"audio"},
		"audio":             map[string]any{"input": input, "output": output},
	}
	if tools := ToolDefinitions(p.Tools, sc.Queues, sc.Topics); len(tools) > 0 {
		session["tools"] = tools
		session["tool_choice"] = "auto"
	}
	return session
}

// Instructions composes the system prompt for a phone conversation.
func Instructions(p domain.AIProfile, sc SessionContext) string {
	var b strings.Builder
	base := strings.TrimSpace(p.Instructions)
	if base == "" {
		base = "You are a friendly, efficient call center voice agent."
	}
	b.WriteString(base)
	b.WriteString("\n\n# Call context\n")
	b.WriteString("You are speaking with a caller on the phone. Keep replies short and natural, one or two sentences. ")
	b.WriteString("Never read out URLs or long lists. Confirm numbers digit by digit.\n")
	if p.Language != "" {
		fmt.Fprintf(&b, "Speak in language code %q unless the caller clearly uses another language.\n", p.Language)
	}
	if sc.CallerNumber != "" {
		fmt.Fprintf(&b, "Caller ID: %s", sc.CallerNumber)
		if sc.CallerName != "" {
			fmt.Fprintf(&b, " (%s)", sc.CallerName)
		}
		b.WriteString(".\n")
	}
	if has(p.Tools, domain.ToolRouteCall) {
		b.WriteString("When the caller needs a person, tell them briefly that you are connecting them, then call ")
		b.WriteString("route_call with their request and a summary; the organization's rules choose the right employee. ")
		b.WriteString("Prefer route_call over transfer_to_queue.\n")
	}
	if has(p.Tools, domain.ToolSearchKnowledge) {
		b.WriteString("Before answering questions about policies, prices, products or procedures, call search_knowledge. ")
		b.WriteString("Answer only from what it returns; if it returns nothing relevant, say you will connect them to someone who knows.\n")
	}
	if has(p.Tools, domain.ToolCreateTicket) {
		b.WriteString("If the request needs follow-up work, call create_ticket and tell the caller their ticket number.\n")
	}
	if has(p.Tools, domain.ToolTicketStatus) || has(p.Tools, domain.ToolUpdateTicket) {
		b.WriteString("If the caller mentions an existing ticket, use ticket_status to check it and update_ticket to add new details.\n")
	}
	if has(p.Tools, domain.ToolTransferToQueue) {
		b.WriteString("If the caller asks for a human, is upset, or you cannot resolve the request, ")
		b.WriteString("briefly tell them you are connecting them, then call transfer_to_queue with a short summary.")
		if len(sc.Queues) > 0 {
			fmt.Fprintf(&b, " Available queues: %s.", strings.Join(sc.Queues, ", "))
		}
		b.WriteString("\n")
	}
	if has(p.Tools, domain.ToolEndCall) {
		b.WriteString("When the caller's needs are met, say a short goodbye and then call end_call.\n")
	}
	if kb := strings.TrimSpace(p.KnowledgeBase); kb != "" {
		b.WriteString("\n# Knowledge base\nUse only these facts when answering factual questions:\n")
		b.WriteString(kb)
		b.WriteString("\n")
	}
	return b.String()
}

func has(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
