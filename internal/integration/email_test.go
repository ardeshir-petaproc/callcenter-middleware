package integration

import (
	"bytes"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"callcenter-middleware/internal/domain"

	"github.com/emersion/go-imap/backend/memory"
	imapserver "github.com/emersion/go-imap/server"
)

// fakeIMAP serves the go-imap memory backend (user "username"/"password").
func fakeIMAP(t *testing.T) (port int, deliver func(raw string)) {
	be := memory.New()
	srv := imapserver.New(be)
	srv.AllowInsecureAuth = true
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { srv.Close() })
	user, _ := be.Login(nil, "username", "password")
	inbox, _ := user.GetMailbox("INBOX")
	return ln.Addr().(*net.TCPAddr).Port, func(raw string) {
		raw = strings.ReplaceAll(raw, "\n", "\r\n")
		if err := inbox.CreateMessage(nil, time.Now(), bytes.NewBufferString(raw)); err != nil {
			t.Fatal(err)
		}
	}
}

func email(id, from, subject, body string, inReplyTo string) string {
	h := fmt.Sprintf("From: %s\nTo: support@acme.test\nSubject: %s\nDate: %s\nMessage-ID: <%s>\n",
		from, subject, time.Now().Format(time.RFC1123Z), id)
	if inReplyTo != "" {
		h += "In-Reply-To: <" + inReplyTo + ">\n"
	}
	return h + "Content-Type: text/plain; charset=utf-8\n\n" + body
}

func TestEmail_ConsentTicketsThreadingAndPurge(t *testing.T) {
	s := newStack(t, 40)
	admin := s.login("admin", "admin123")
	port, deliver := fakeIMAP(t)
	smtpPort, sent := fakeSMTP(t)
	conn := map[string]any{"name": "Support inbox", "host": "127.0.0.1", "port": port, "security": "none",
		"username": "username", "password": "password", "from_filter": []string{"@customer.com"}, "ticket_category": "email",
		"smtp_host": "127.0.0.1", "smtp_port": smtpPort, "smtp_security": "none", "from_address": "support@acme.test"}
	if code := s.call("POST", "/email-connectors", admin, conn, nil); code != 400 {
		t.Fatalf("connecting without consent must fail, got %d", code)
	}
	conn["consent"] = true
	var c domain.EmailConnector
	if code := s.call("POST", "/email-connectors", admin, conn, &c); code != 201 || c.Password != "" || !c.HasPassword {
		t.Fatalf("connect: %d %+v", code, c)
	}
	var folders []string
	s.call("POST", "/email-connectors/folders", admin, map[string]any{"id": c.ID, "host": "127.0.0.1", "port": port,
		"security": "none", "username": "username"}, &folders)
	if len(folders) == 0 || folders[0] != "INBOX" {
		t.Fatalf("folders: %v", folders)
	}

	deliver(email("m1@customer.com", "Ali <ali@customer.com>", "Invoice problem", "My invoice total is wrong.", ""))
	deliver(email("x1@other.com", "spam@other.com", "Buy now", "offer", ""))
	var res struct{ Imported int }
	s.call("POST", "/email-connectors/"+itoa(c.ID)+"/sync", admin, nil, &res)
	if res.Imported != 1 {
		t.Fatalf("filters: only the customer mail should be imported, got %d", res.Imported)
	}
	var list struct {
		Items []domain.Ticket `json:"items"`
	}
	s.call("GET", "/tickets?q=Invoice+problem", admin, nil, &list)
	if len(list.Items) != 1 || list.Items[0].Source != "email" || list.Items[0].RequesterEmail != "ali@customer.com" {
		t.Fatalf("ticket from email: %+v", list.Items)
	}
	tk := list.Items[0]

	// A reply threads onto the same ticket without the quoted history.
	deliver(email("m2@customer.com", "ali@customer.com", "Re: Invoice problem", "Any update?\n> My invoice total is wrong.", "m1@customer.com"))
	s.call("POST", "/email-connectors/"+itoa(c.ID)+"/sync", admin, nil, &res)
	var detail struct {
		Comments []domain.TicketComment `json:"comments"`
	}
	s.call("GET", "/tickets/"+itoa(tk.ID), admin, nil, &detail)
	if res.Imported != 1 || len(detail.Comments) != 1 || detail.Comments[0].Body != "Any update?" {
		t.Fatalf("threaded reply: %d %+v", res.Imported, detail.Comments)
	}
	s.call("POST", "/email-connectors/"+itoa(c.ID)+"/sync", admin, nil, &res)
	if res.Imported != 0 {
		t.Fatalf("re-sync must not import twice, got %d", res.Imported)
	}
	// Quoting someone else's ticket number does not write into that ticket.
	deliver(email("m3@customer.com", "mallory@customer.com", "Re: ["+tk.Number+"] hijack", "attacker text", ""))
	s.call("POST", "/email-connectors/"+itoa(c.ID)+"/sync", admin, nil, &res)
	s.call("GET", "/tickets/"+itoa(tk.ID), admin, nil, &detail)
	if len(detail.Comments) != 1 {
		t.Fatalf("spoofed sender must not join the ticket: %+v", detail.Comments)
	}
	s.call("GET", "/tickets?q=hijack", admin, nil, &list)
	if len(list.Items) != 1 || list.Items[0].RequesterEmail != "mallory@customer.com" {
		t.Fatalf("spoofed reply should open its own ticket: %+v", list.Items)
	}
	// A staff reply is emailed to the customer with the ticket number in
	// the subject; internal notes are not.
	s.call("POST", "/tickets/"+itoa(tk.ID)+"/comments", admin, map[string]any{"body": "internal only", "internal": true}, nil)
	s.call("POST", "/tickets/"+itoa(tk.ID)+"/comments", admin, map[string]any{"body": "We corrected the invoice."}, nil)
	select {
	case msg := <-sent:
		if !strings.Contains(msg, "To: ali@customer.com") || !strings.Contains(msg, tk.Number) ||
			!strings.Contains(msg, "In-Reply-To: <m2@customer.com>") || strings.Contains(msg, "internal only") {
			t.Fatalf("reply email: %s", msg)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("reply was not emailed")
	}

	// Knowledge from mail lands as draft; the AI can be cut off from it.
	var kc domain.EmailConnector
	s.call("POST", "/email-connectors", admin, map[string]any{"host": "127.0.0.1", "port": port, "security": "none",
		"username": "username", "password": "password", "mode": "kb", "subject_filter": "Policy", "ai_enabled": true,
		"consent": true}, &kc)
	deliver(email("p1@customer.com", "ops@customer.com", "Policy: warranty", "Warranty lasts twenty four months.", ""))
	s.call("POST", "/email-connectors/"+itoa(kc.ID)+"/sync", admin, nil, &res)
	var hits []domain.KBHit
	s.call("POST", "/kb/search/ai", admin, map[string]any{"q": "warranty"}, &hits)
	if res.Imported != 1 || len(hits) != 0 {
		t.Fatalf("draft email knowledge must not reach the AI: %d %+v", res.Imported, hits)
	}
	var sources []domain.KnowledgeSource
	s.call("GET", "/kb/sources?q=warranty", admin, nil, &sources)
	if len(sources) != 1 || sources[0].Kind != "email" || sources[0].Status != "draft" {
		t.Fatalf("email source: %+v", sources)
	}
	src := sources[0]
	src.Status = "active"
	s.call("PUT", "/kb/sources/"+itoa(src.ID), admin, src, nil)
	s.call("POST", "/kb/search/ai", admin, map[string]any{"q": "warranty"}, &hits)
	if len(hits) != 1 {
		t.Fatalf("approved email knowledge should be searchable: %+v", hits)
	}
	s.call("POST", "/email-connectors/"+itoa(kc.ID)+"/ai", admin, map[string]any{"enabled": false}, nil)
	s.call("POST", "/kb/search/ai", admin, map[string]any{"q": "warranty"}, &hits)
	if len(hits) != 0 {
		t.Fatalf("excluded mailbox still used by AI: %+v", hits)
	}

	// Access control and tenant isolation.
	if code := s.call("GET", "/email-connectors", s.login("alice", "agent123"), nil, nil); code != 403 {
		t.Fatalf("agent access: %d", code)
	}
	var other []domain.EmailConnector
	s.call("GET", "/email-connectors", s.loginTenant("admin", "admin123", "parsian"), nil, &other)
	if len(other) != 0 {
		t.Fatalf("other tenant sees connectors: %+v", other)
	}
	// Disconnect with purge removes credentials and imported knowledge.
	s.call("DELETE", "/email-connectors/"+itoa(kc.ID)+"?purge=1", admin, nil, nil)
	if code := s.call("GET", "/kb/sources/"+itoa(src.ID), admin, nil, nil); code != 404 {
		t.Fatalf("purged source still present: %d", code)
	}
	var audit []domain.AuditEntry
	s.call("GET", "/audit?action=email.disconnect", admin, nil, &audit)
	if len(audit) != 1 {
		t.Fatalf("disconnect audit: %+v", audit)
	}
}
