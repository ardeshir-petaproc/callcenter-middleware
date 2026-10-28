package mail

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/kb"
	"callcenter-middleware/internal/secure"
	"callcenter-middleware/internal/store"
	"callcenter-middleware/internal/tickets"
)

// batch bounds how many messages one sync imports.
const batch = 50

var ticketTag = regexp.MustCompile(`\[(T-\d+)\]`)

// Service imports mail from consented connectors into tickets or the
// knowledge base.
type Service struct {
	Store   *store.Store
	Box     *secure.Box
	Tickets *tickets.Service
	KB      *kb.Service
	Mailbox Mailbox
	Now     func() time.Time
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Password decrypts a connector's stored credential.
func (s *Service) Password(c *domain.EmailConnector) (string, error) {
	return s.Box.OpenString(c.TenantID, c.Password)
}

// Sync imports new messages for one connector (ctx must be tenant scoped)
// and returns how many were imported.
func (s *Service) Sync(ctx context.Context, id int64) (int, error) {
	c, err := s.Store.Mailboxes.Get(ctx, id)
	if err != nil {
		return 0, err
	}
	n, err := s.sync(ctx, c)
	c.LastSync, c.Status, c.LastError = s.now().UTC().Format(time.RFC3339), "ok", ""
	if err != nil {
		c.Status, c.LastError = "error", err.Error()
	}
	c.Imported += n
	if uerr := s.Store.Mailboxes.Update(ctx, c); uerr != nil && err == nil {
		err = uerr
	}
	return n, err
}

func (s *Service) sync(ctx context.Context, c *domain.EmailConnector) (int, error) {
	pw, err := s.Password(c)
	if err != nil {
		return 0, fmt.Errorf("credentials unavailable: %w", err)
	}
	var since time.Time
	if c.SinceDate != "" {
		since, _ = time.Parse("2006-01-02", c.SinceDate)
	}
	validity, raws, err := s.Mailbox.Fetch(ctx, c, pw, c.LastUID, since, batch)
	if err != nil {
		return 0, err
	}
	if c.UIDValidity != 0 && validity != c.UIDValidity {
		c.LastUID = 0 // the folder was recreated; UIDs restart
	}
	c.UIDValidity = validity
	imported := 0
	for _, raw := range raws {
		if raw.UID > c.LastUID {
			c.LastUID = raw.UID
		}
		p, err := Parse(raw.Data)
		if err != nil || !Matches(c, p) {
			continue
		}
		ok, err := s.ingest(ctx, c, raw.UID, p)
		if err != nil {
			return imported, err
		}
		if ok {
			imported++
		}
	}
	return imported, nil
}

// Matches applies the connector's sender and subject filters.
func Matches(c *domain.EmailConnector, p *Parsed) bool {
	if c.SubjectFilter != "" && !strings.Contains(strings.ToLower(p.Subject), strings.ToLower(c.SubjectFilter)) {
		return false
	}
	if len(c.FromFilter) == 0 {
		return true
	}
	for _, f := range c.FromFilter {
		f = strings.ToLower(strings.TrimSpace(f))
		if f == p.FromAddr || (strings.HasPrefix(f, "@") && strings.HasSuffix(p.FromAddr, f)) {
			return true
		}
	}
	return false
}

func (s *Service) ingest(ctx context.Context, c *domain.EmailConnector, uid uint32, p *Parsed) (bool, error) {
	body, err := s.Box.SealString(c.TenantID, p.Text)
	if err != nil {
		return false, err
	}
	m := &domain.EmailMessage{ConnectorID: c.ID, UID: uid, MessageID: p.MessageID, InReplyTo: p.References,
		FromAddr: p.FromAddr, FromName: p.FromName, Subject: p.Subject, Date: p.Date, Body: body}
	if m.InReplyTo == nil {
		m.InReplyTo = []string{}
	}
	isNew, err := s.Store.SaveEmail(ctx, m)
	if err != nil || !isNew {
		return false, err
	}
	text := p.Text
	if len(p.Attachments) > 0 {
		text += "\n\n[Attachments: " + strings.Join(p.Attachments, ", ") + "]"
	}
	if c.Mode == domain.EmailToKB {
		src := &domain.KnowledgeSource{Title: firstNonEmpty(p.Subject, "Email from "+p.FromAddr), Kind: domain.KBEmail,
			Category: c.KBCategory, AIEnabled: c.AIEnabled, Status: domain.KBDraft, ConnectorID: c.ID, EmailID: m.ID,
			OwnerID: c.ConsentBy}
		if c.DepartmentID > 0 {
			src.DepartmentIDs = []int64{c.DepartmentID}
		}
		if err := s.KB.Save(ctx, src, "", []byte(text)); err != nil {
			return false, err
		}
		return true, s.Store.LinkEmail(ctx, m.ID, 0, src.ID)
	}
	ticketID, err := s.toTicket(ctx, c, p, text)
	if err != nil {
		return false, err
	}
	return true, s.Store.LinkEmail(ctx, m.ID, ticketID, 0)
}
