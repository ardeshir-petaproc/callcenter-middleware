package mail

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"log/slog"
	"mime"
	"net"
	"net/smtp"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/tickets"

	"github.com/google/uuid"
)

// Reply emails a staff reply to the customer of a ticket that came in by
// email, through the connector the customer wrote to. The subject carries
// the ticket number so the customer's answer threads back. Delivery
// failures are recorded on the ticket as an internal note.
func (s *Service) Reply(ctx context.Context, t *domain.Ticket, c *domain.TicketComment) {
	if t.RequesterEmail == "" {
		return
	}
	connID, inReplyTo, err := s.Store.TicketMailbox(ctx, t.ID)
	if err != nil {
		return // not an email ticket
	}
	conn, err := s.Store.Mailboxes.Get(ctx, connID)
	if err != nil || conn.SMTPHost == "" {
		return
	}
	if err := s.send(conn, t, c.Body, inReplyTo); err != nil {
		slog.Warn("Ticket email reply failed", "ticket", t.Number, "err", err)
		_, _ = s.Tickets.Comment(ctx, t.ID, "Email delivery to "+t.RequesterEmail+" failed: "+err.Error(), true,
			tickets.Actor{Name: "Email"})
	}
}

func (s *Service) send(conn *domain.EmailConnector, t *domain.Ticket, body, inReplyTo string) error {
	pw, err := s.Password(conn)
	if err != nil {
		return err
	}
	from := firstNonEmpty(conn.FromAddress, conn.Username)
	msg := compose(from, t, body, inReplyTo)
	addr := net.JoinHostPort(conn.SMTPHost, fmt.Sprint(conn.SMTPPort))
	var nc net.Conn
	d := &net.Dialer{Timeout: 20 * time.Second}
	if conn.SMTPSecurity == domain.EmailTLS {
		nc, err = tls.DialWithDialer(d, "tcp", addr, &tls.Config{ServerName: conn.SMTPHost})
	} else {
		nc, err = d.Dial("tcp", addr)
	}
	if err != nil {
		return err
	}
	_ = nc.SetDeadline(time.Now().Add(time.Minute))
	cl, err := smtp.NewClient(nc, conn.SMTPHost)
	if err != nil {
		nc.Close()
		return err
	}
	defer cl.Close()
	if conn.SMTPSecurity == domain.EmailStartTLS {
		if err := cl.StartTLS(&tls.Config{ServerName: conn.SMTPHost}); err != nil {
			return err
		}
	}
	if ok, _ := cl.Extension("AUTH"); ok {
		if err := cl.Auth(smtp.PlainAuth("", conn.Username, pw, conn.SMTPHost)); err != nil {
			return err
		}
	}
	if err := cl.Mail(from); err != nil {
		return err
	}
	if err := cl.Rcpt(t.RequesterEmail); err != nil {
		return err
	}
	w, err := cl.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return cl.Quit()
}

// compose builds a UTF-8 plain-text message (base64 body, so Persian text
// survives any relay).
func compose(from string, t *domain.Ticket, body, inReplyTo string) []byte {
	var b bytes.Buffer
	subject := fmt.Sprintf("Re: [%s] %s", t.Number, strings.TrimPrefix(t.Subject, "Re: "))
	fmt.Fprintf(&b, "From: %s\r\nTo: %s\r\nSubject: %s\r\n", from, t.RequesterEmail, mime.QEncoding.Encode("utf-8", subject))
	fmt.Fprintf(&b, "Date: %s\r\nMessage-ID: <%s@%s>\r\n", time.Now().Format(time.RFC1123Z), uuid.NewString(), domainOf(from))
	if inReplyTo != "" {
		fmt.Fprintf(&b, "In-Reply-To: <%s>\r\nReferences: <%s>\r\n", inReplyTo, inReplyTo)
	}
	b.WriteString("MIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\nContent-Transfer-Encoding: base64\r\n\r\n")
	enc := base64.StdEncoding.EncodeToString([]byte(body))
	for len(enc) > 76 {
		b.WriteString(enc[:76] + "\r\n")
		enc = enc[76:]
	}
	b.WriteString(enc + "\r\n")
	return b.Bytes()
}

func domainOf(addr string) string {
	if i := strings.LastIndex(addr, "@"); i >= 0 {
		return strings.Trim(addr[i+1:], "> ")
	}
	return "callcenter.local"
}
