package mail

import (
	"bytes"
	"io"
	"strings"
	"time"

	"callcenter-middleware/internal/kb"

	_ "github.com/emersion/go-message/charset" // decode legacy charsets such as windows-1256
	"github.com/emersion/go-message/mail"
)

// Parsed is the part of a message the connector uses.
type Parsed struct {
	MessageID   string
	References  []string // In-Reply-To first, then References
	FromAddr    string
	FromName    string
	Subject     string
	Date        time.Time
	Text        string
	Attachments []string
}

// Parse extracts headers, the readable body (text/plain preferred, HTML
// converted to text) and attachment names.
func Parse(raw []byte) (*Parsed, error) {
	r, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil && r == nil {
		return nil, err
	}
	h := r.Header
	p := &Parsed{}
	p.MessageID, _ = h.MessageID()
	if ids, err := h.MsgIDList("In-Reply-To"); err == nil {
		p.References = append(p.References, ids...)
	}
	if ids, err := h.MsgIDList("References"); err == nil {
		for i := len(ids) - 1; i >= 0; i-- {
			p.References = append(p.References, ids[i])
		}
	}
	if from, err := h.AddressList("From"); err == nil && len(from) > 0 {
		p.FromAddr, p.FromName = strings.ToLower(from[0].Address), from[0].Name
	}
	p.Subject, _ = h.Subject()
	if p.Date, err = h.Date(); err != nil || p.Date.IsZero() {
		p.Date = time.Now().UTC()
	}
	var plain, html string
	for {
		part, err := r.NextPart()
		if err == io.EOF || part == nil {
			break
		}
		if err != nil {
			continue
		}
		switch ph := part.Header.(type) {
		case *mail.InlineHeader:
			ct, _, _ := ph.ContentType()
			data, _ := io.ReadAll(io.LimitReader(part.Body, 5<<20))
			if ct == "text/html" && html == "" {
				html = string(data)
			} else if (ct == "text/plain" || ct == "") && plain == "" {
				plain = string(data)
			}
		case *mail.AttachmentHeader:
			name, _ := ph.Filename()
			p.Attachments = append(p.Attachments, name)
		}
	}
	p.Text = strings.TrimSpace(plain)
	if p.Text == "" && html != "" {
		p.Text, _ = kb.Extract("body.html", []byte(html))
	}
	p.Text = stripQuoted(p.Text)
	return p, nil
}

// stripQuoted drops the quoted history of a reply so tickets show only the
// new text.
func stripQuoted(s string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if i > 0 && strings.HasPrefix(t, "On ") && strings.HasSuffix(t, "wrote:") {
			return strings.TrimSpace(strings.Join(lines[:i], "\n"))
		}
		if strings.HasPrefix(t, "-----Original Message-----") {
			return strings.TrimSpace(strings.Join(lines[:i], "\n"))
		}
	}
	out := lines[:0]
	for _, l := range lines {
		if !strings.HasPrefix(strings.TrimSpace(l), ">") {
			out = append(out, l)
		}
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}
