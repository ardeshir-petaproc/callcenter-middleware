// Package mail connects consented mailboxes over IMAP. Only the configured
// folder is read, starting from the configured date, and only messages
// passing the sender/subject filters are imported.
package mail

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"

	"callcenter-middleware/internal/domain"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// Mailbox reads one folder of a mailbox.
type Mailbox interface {
	Folders(ctx context.Context, c *domain.EmailConnector, password string) ([]string, error)
	// Fetch returns up to max messages with UID > afterUID (raw RFC 822)
	// and the folder's UIDVALIDITY.
	Fetch(ctx context.Context, c *domain.EmailConnector, password string, afterUID uint32, since time.Time, max int) (uint32, []Raw, error)
}

// Raw is one fetched message.
type Raw struct {
	UID  uint32
	Data []byte
}

// IMAP is the network implementation of Mailbox.
type IMAP struct{ Timeout time.Duration }

func (m IMAP) dial(c *domain.EmailConnector, password string) (*client.Client, error) {
	addr := net.JoinHostPort(c.Host, fmt.Sprint(c.Port))
	timeout := m.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	dialer := &net.Dialer{Timeout: timeout}
	var cl *client.Client
	var err error
	if c.Security == domain.EmailTLS {
		cl, err = client.DialWithDialerTLS(dialer, addr, &tls.Config{ServerName: c.Host})
	} else {
		cl, err = client.DialWithDialer(dialer, addr)
	}
	if err != nil {
		return nil, err
	}
	cl.Timeout = timeout
	if c.Security == domain.EmailStartTLS {
		if err := cl.StartTLS(&tls.Config{ServerName: c.Host}); err != nil {
			cl.Logout()
			return nil, err
		}
	}
	if err := cl.Login(c.Username, password); err != nil {
		cl.Logout()
		return nil, fmt.Errorf("login failed: %w", err)
	}
	return cl, nil
}

// Folders lists mailbox folders so an administrator can pick one.
func (m IMAP) Folders(ctx context.Context, c *domain.EmailConnector, password string) ([]string, error) {
	cl, err := m.dial(c, password)
	if err != nil {
		return nil, err
	}
	defer cl.Logout()
	ch := make(chan *imap.MailboxInfo, 32)
	done := make(chan error, 1)
	go func() { done <- cl.List("", "*", ch) }()
	out := []string{}
	for mb := range ch {
		out = append(out, mb.Name)
	}
	return out, <-done
}

// Fetch reads new messages from the configured folder (read-only select,
// so flags such as \Seen are left untouched).
func (m IMAP) Fetch(ctx context.Context, c *domain.EmailConnector, password string, afterUID uint32, since time.Time, max int) (uint32, []Raw, error) {
	cl, err := m.dial(c, password)
	if err != nil {
		return 0, nil, err
	}
	defer cl.Logout()
	status, err := cl.Select(c.Folder, true)
	if err != nil {
		return 0, nil, fmt.Errorf("open folder %q: %w", c.Folder, err)
	}
	crit := imap.NewSearchCriteria()
	crit.Uid = new(imap.SeqSet)
	crit.Uid.AddRange(afterUID+1, 0)
	if !since.IsZero() {
		crit.Since = since
	}
	uids, err := cl.UidSearch(crit)
	if err != nil {
		return 0, nil, err
	}
	set := new(imap.SeqSet)
	n := 0
	for _, uid := range uids {
		if uid > afterUID && n < max {
			set.AddNum(uid)
			n++
		}
	}
	if n == 0 {
		return status.UidValidity, nil, nil
	}
	section := &imap.BodySectionName{Peek: true}
	ch := make(chan *imap.Message, 16)
	done := make(chan error, 1)
	go func() { done <- cl.UidFetch(set, []imap.FetchItem{imap.FetchUid, section.FetchItem()}, ch) }()
	var out []Raw
	for msg := range ch {
		if body := msg.GetBody(section); body != nil {
			data, _ := io.ReadAll(io.LimitReader(body, 30<<20))
			out = append(out, Raw{UID: msg.Uid, Data: data})
		}
	}
	return status.UidValidity, out, <-done
}
