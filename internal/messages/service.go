// Package messages turns customer chat messages (Joow conversations or any
// other channel posting to the inbound endpoint) into tickets, threads
// follow-ups onto them and sends staff replies back to the conversation.
package messages

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/joow"
	"callcenter-middleware/internal/store"
	"callcenter-middleware/internal/tickets"
)

// ErrInvalid reports an unusable inbound message.
var ErrInvalid = errors.New("invalid message")

// Service handles inbound and outbound chat messages.
type Service struct {
	Store   *store.Store
	Tickets *tickets.Service
	Joow    joow.Provider
}

// Receive records a customer message (ctx must be tenant scoped) and
// returns the ticket it belongs to and whether the ticket is new.
func (s *Service) Receive(ctx context.Context, m domain.InboundMessage) (*domain.Ticket, bool, error) {
	m.Channel, m.Text = strings.ToLower(strings.TrimSpace(m.Channel)), strings.TrimSpace(m.Text)
	if m.Channel == "" || m.ConversationID == "" || m.Text == "" {
		return nil, false, fmt.Errorf("%w: channel, conversation_id and text are required", ErrInvalid)
	}
	who := tickets.Actor{Name: firstNonEmpty(m.FromName, m.FromID, "Customer")}
	if conv, err := s.Store.FindConversation(ctx, m.Channel, m.ConversationID); err == nil {
		if t, err := s.Store.GetTicket(ctx, conv.TicketID); err == nil && t.Status != domain.TicketClosed {
			if _, err := s.Tickets.Comment(ctx, t.ID, m.Text, false, who); err != nil {
				return nil, false, err
			}
			if t.Status == domain.TicketWaiting || t.Status == domain.TicketResolved {
				st := domain.TicketInProgress
				if t, err = s.Tickets.Update(ctx, t.ID, tickets.Patch{Status: &st}, who); err != nil {
					return nil, false, err
				}
			}
			return t, false, nil
		}
	}
	subject := []rune(strings.SplitN(m.Text, "\n", 2)[0])
	if len(subject) > 80 {
		subject = append(subject[:80], '…')
	}
	t := &domain.Ticket{Subject: string(subject), Description: m.Text, Source: "message", Category: m.Channel,
		RequesterName: m.FromName, RequesterPhone: m.FromPhone, RequesterEmail: m.FromEmail}
	if c := core.FindContact(ctx, s.Store, m.FromPhone); m.FromPhone != "" && c != nil {
		t.ContactID = c.ID
	}
	if err := s.Tickets.Create(ctx, t, who); err != nil {
		return nil, false, err
	}
	err := s.Store.SaveConversation(ctx, &domain.Conversation{Channel: m.Channel, ExternalID: m.ConversationID, TicketID: t.ID})
	return t, true, err
}

// Reply sends a staff reply on a chat ticket back to its conversation.
// Failures are noted on the ticket.
func (s *Service) Reply(ctx context.Context, t *domain.Ticket, c *domain.TicketComment) {
	conv, err := s.Store.TicketConversation(ctx, t.ID)
	if err != nil || conv.Channel != "joow" {
		return
	}
	msgr, ok := s.Joow.(joow.Messenger)
	if !ok {
		return
	}
	if err := msgr.SendMessage(ctx, conv.ExternalID, c.Body); err != nil {
		_, _ = s.Tickets.Comment(ctx, t.ID, "Joow message delivery failed: "+err.Error(), true, tickets.Actor{Name: "Joow"})
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
