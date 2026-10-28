// Package joow integrates with the Joow platform: identity (employees keep
// their Joow accounts), account lookup for invitations, and notifications
// (invites, incoming-call alerts). The real Joow APIs are reached through
// the HTTP provider; the local provider serves development and demos.
package joow

import (
	"context"
	"errors"
)

// ErrInvalidToken is returned when a Joow session token cannot be verified.
var ErrInvalidToken = errors.New("joow: invalid token")

// ErrNotFound is returned for unknown accounts.
var ErrNotFound = errors.New("joow: account not found")

// Account is a Joow user account.
type Account struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	Phone     string `json:"phone"`
	AvatarURL string `json:"avatar_url"`
}

// Notification is pushed to a Joow account (in-app/push notification).
type Notification struct {
	Kind  string         `json:"kind"` // invitation, incoming_call, ticket_assigned, ...
	Title string         `json:"title"`
	Body  string         `json:"body"`
	URL   string         `json:"url"`
	Data  map[string]any `json:"data,omitempty"`
}

// Messenger is implemented by providers that can send chat messages into a
// Joow conversation (replies to customers who wrote via Joow).
type Messenger interface {
	SendMessage(ctx context.Context, conversationID, text string) error
}

// Provider is the boundary to Joow.
type Provider interface {
	Name() string
	// VerifyToken validates a Joow session token (from the mini-app or the
	// main Joow app) and returns the signed-in account.
	VerifyToken(ctx context.Context, token string) (*Account, error)
	// Search finds accounts by name, email or phone for invitations.
	Search(ctx context.Context, query string, limit int) ([]Account, error)
	// Get loads an account by id.
	Get(ctx context.Context, id string) (*Account, error)
	// Notify sends a notification to an account.
	Notify(ctx context.Context, accountID string, n Notification) error
}
