package server

import (
	"net/http"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"
)

func normalizeMailbox(c *domain.EmailConnector) error {
	c.Host = strings.TrimSpace(c.Host)
	if c.Host == "" || c.Username == "" {
		return bad("host and username are required")
	}
	switch c.Security {
	case "":
		c.Security = domain.EmailTLS
	case domain.EmailTLS, domain.EmailStartTLS, domain.EmailPlain:
	default:
		return bad("security must be tls, starttls or none")
	}
	if c.Port == 0 {
		c.Port = 993
		if c.Security != domain.EmailTLS {
			c.Port = 143
		}
	}
	if c.Folder == "" {
		c.Folder = "INBOX"
	}
	return nil
}

func validateMailbox(c *domain.EmailConnector) error {
	if err := normalizeMailbox(c); err != nil {
		return err
	}
	if strings.TrimSpace(c.Name) == "" {
		c.Name = c.Username
	}
	switch c.Mode {
	case "":
		c.Mode = domain.EmailToTicket
	case domain.EmailToTicket, domain.EmailToKB:
	default:
		return bad("mode must be ticket or kb")
	}
	if c.PollMinutes <= 0 {
		c.PollMinutes = 5
	}
	if c.PollMinutes > 1440 {
		return bad("poll_minutes must be at most 1440")
	}
	if c.SinceDate != "" {
		if _, err := time.Parse("2006-01-02", c.SinceDate); err != nil {
			return bad("since_date must be YYYY-MM-DD")
		}
	}
	if c.FromFilter == nil {
		c.FromFilter = []string{}
	}
	return normalizeSMTP(c)
}

// normalizeSMTP validates the optional outgoing mail settings used to send
// ticket replies to customers.
func normalizeSMTP(c *domain.EmailConnector) error {
	if c.SMTPHost = strings.TrimSpace(c.SMTPHost); c.SMTPHost == "" {
		return nil
	}
	ports := map[string]int{domain.EmailTLS: 465, domain.EmailStartTLS: 587, domain.EmailPlain: 25}
	if c.SMTPSecurity == "" {
		c.SMTPSecurity = domain.EmailStartTLS
	}
	if _, ok := ports[c.SMTPSecurity]; !ok {
		return bad("smtp_security must be tls, starttls or none")
	}
	if c.SMTPPort == 0 {
		c.SMTPPort = ports[c.SMTPSecurity]
	}
	if strings.ContainsAny(c.FromAddress, "<> ") {
		return bad("from_address must be a plain email address")
	}
	return nil
}

// saveMailbox creates (with explicit consent) or updates a connector.
// The password is sealed; an empty password on update keeps the old one.
// Changing the mailbox or folder restarts the import position.
func (s *Server) saveMailbox(w http.ResponseWriter, r *http.Request) error {
	var req struct {
		domain.EmailConnector
		Consent bool `json:"consent"`
	}
	if err := decode(r, &req); err != nil {
		return err
	}
	c := req.EmailConnector
	if err := validateMailbox(&c); err != nil {
		return err
	}
	if err := s.exists(r, "department", c.DepartmentID); err != nil {
		return err
	}
	tenant := tenantOf(r)
	c.TenantID, c.HasPassword = 0, false
	if c.Password != "" {
		sealed, err := s.d.Mail.Box.SealString(tenant, c.Password)
		if err != nil {
			return err
		}
		c.Password = sealed
	}
	id, _ := pathID(r)
	if id == 0 {
		if !req.Consent {
			return bad("explicit consent is required to connect a mailbox")
		}
		if c.Password == "" {
			return bad("password is required")
		}
		c.ID, c.ConsentBy, c.ConsentAt, c.Status = 0, claimsOf(r).UserID, nowRFC(), "never"
		c.LastUID, c.UIDValidity, c.Imported, c.LastSync, c.LastError = 0, 0, 0, "", ""
		if err := s.d.Store.Mailboxes.Create(r.Context(), &c); err != nil {
			return err
		}
		s.audit(r, "email.connect", "email_connector", c.ID, c.Username+" / "+c.Folder)
		writeJSON(w, http.StatusCreated, publicMailbox(c))
		return nil
	}
	prev, err := s.d.Store.Mailboxes.Get(r.Context(), id)
	if err != nil {
		return err
	}
	if c.Password == "" {
		c.Password = prev.Password
	}
	c.ID, c.ConsentBy, c.ConsentAt, c.Status, c.LastSync, c.LastError = id, prev.ConsentBy, prev.ConsentAt, prev.Status, prev.LastSync, prev.LastError
	c.LastUID, c.UIDValidity, c.Imported = prev.LastUID, prev.UIDValidity, prev.Imported
	if c.Host != prev.Host || c.Username != prev.Username || c.Folder != prev.Folder {
		c.LastUID, c.UIDValidity = 0, 0
	}
	if err := s.d.Store.Mailboxes.Update(r.Context(), &c); err != nil {
		return err
	}
	s.audit(r, "email.update", "email_connector", c.ID, c.Username+" / "+c.Folder)
	writeJSON(w, http.StatusOK, publicMailbox(c))
	return nil
}
