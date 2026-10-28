package domain

import "time"

// InboundMessage is a customer message from a chat channel (e.g. Joow).
type InboundMessage struct {
	Channel        string `json:"channel"`         // e.g. "joow"
	ConversationID string `json:"conversation_id"` // the channel's thread id
	FromID         string `json:"from_id"`
	FromName       string `json:"from_name"`
	FromPhone      string `json:"from_phone"`
	FromEmail      string `json:"from_email"`
	Text           string `json:"text"`
}

// Conversation links a chat thread to the ticket it became.
type Conversation struct {
	ID         int64     `json:"id"`
	TenantID   int64     `json:"tenant_id"`
	Channel    string    `json:"channel"`
	ExternalID string    `json:"external_id"`
	TicketID   int64     `json:"ticket_id"`
	CreatedAt  time.Time `json:"created_at"`
}
