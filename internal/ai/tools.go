package ai

import (
	"callcenter-middleware/internal/domain"
)

func obj(props map[string]any, required ...string) map[string]any {
	o := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		o["required"] = required
	}
	return o
}

func str(desc string) map[string]any { return map[string]any{"type": "string", "description": desc} }

// ToolDefinitions returns the Realtime function tools enabled for a profile.
func ToolDefinitions(enabled []string, queues, topics []string) []map[string]any {
	var out []map[string]any
	add := func(name, desc string, params map[string]any) {
		if has(enabled, name) {
			out = append(out, map[string]any{"type": "function", "name": name, "description": desc, "parameters": params})
		}
	}
	queueParam := str("Name of the queue to transfer to.")
	if len(queues) > 0 {
		queueParam["enum"] = queues
	}
	add(domain.ToolTransferToQueue, "Transfer the caller to a human agent queue.", obj(map[string]any{
		"queue":   queueParam,
		"reason":  str("Why the caller is being transferred."),
		"summary": str("One or two sentence summary of the conversation for the human agent."),
	}, "summary"))
	add(domain.ToolEndCall, "Hang up after the conversation is complete and you have said goodbye.", obj(map[string]any{
		"summary":    str("Summary of what was resolved."),
		"resolution": str("resolved, unresolved or caller_left"),
	}, "summary"))
	add(domain.ToolCreateCallback, "Schedule a callback from a human agent.", obj(map[string]any{
		"number":         str("Phone number to call back; omit to use the caller ID."),
		"preferred_time": str("When the caller prefers to be called, in their words."),
		"notes":          str("What the callback is about."),
	}, "notes"))
	add(domain.ToolLookupContact, "Look up the caller's customer record and recent call history.", obj(map[string]any{
		"phone": str("Phone number to look up; omit to use the caller ID."),
	}))
	add(domain.ToolTakeMessage, "Record a message for the team when nobody can help right now.", obj(map[string]any{
		"message":         str("The caller's message."),
		"callback_number": str("Number to reach the caller."),
		"name":            str("Caller's name."),
	}, "message"))
	topicParam := str("The kind of request, from the organization's topics.")
	if len(topics) > 0 {
		topicParam["enum"] = topics
	}
	add(domain.ToolRouteCall, "Connect the caller to the right employee or team. The organization's routing rules, "+
		"job descriptions and live availability decide who; you only describe the request.", obj(map[string]any{
		"topic":   topicParam,
		"request": str("What the caller needs, in their own words."),
		"summary": str("One or two sentence summary for the employee who takes the call."),
	}, "request", "summary"))
	add(domain.ToolSearchKnowledge, "Search the organization's approved knowledge base before answering a factual question.",
		obj(map[string]any{"query": str("What to look up.")}, "query"))
	add(domain.ToolCreateTicket, "Open a support ticket so the team follows up on the caller's request.", obj(map[string]any{
		"subject":     str("Short title of the request."),
		"description": str("Details the team needs, including what was already tried."),
		"priority":    map[string]any{"type": "string", "enum": []string{"low", "normal", "high", "urgent"}},
		"topic":       str("The kind of request."),
	}, "subject", "description"))
	add(domain.ToolTicketStatus, "Look up the status of the caller's existing ticket by its number (e.g. T-1001).",
		obj(map[string]any{"number": str("Ticket number.")}, "number"))
	add(domain.ToolUpdateTicket, "Add new information from the caller to their existing ticket.", obj(map[string]any{
		"number":  str("Ticket number."),
		"message": str("What the caller wants to add, in a sentence or two."),
	}, "number", "message"))
	add(domain.ToolCheckHours, "Check whether the call center is currently open.", obj(map[string]any{}))
	return out
}

// AllTools lists every tool name, for UI defaults.
var AllTools = []string{
	domain.ToolTransferToQueue, domain.ToolEndCall, domain.ToolCreateCallback,
	domain.ToolLookupContact, domain.ToolTakeMessage, domain.ToolCheckHours, domain.ToolRouteCall,
	domain.ToolSearchKnowledge, domain.ToolCreateTicket, domain.ToolTicketStatus, domain.ToolUpdateTicket,
}
