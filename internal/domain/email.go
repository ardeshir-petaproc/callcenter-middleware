package domain

import "time"

// Email connector modes and security settings.
const (
	EmailToTicket = "ticket" // each new conversation opens a ticket, replies are threaded
	EmailToKB     = "kb"     // messages become knowledge sources (draft until reviewed)

	EmailTLS      = "tls"
	EmailStartTLS = "starttls"
	EmailPlain    = "none" // only for local test servers
)

// EmailConnector is an explicitly consented, scoped mailbox connection:
// one folder, optional sender/subject filters and a start date. The full
// mailbox is never read. Credentials are encrypted at rest and never
// returned by the API.
type EmailConnector struct {
	ID             int64    `json:"id"`
	TenantID       int64    `json:"tenant_id"`
	Name           string   `json:"name"`
	Host           string   `json:"host"`
	Port           int      `json:"port"`
	Security       string   `json:"security"`
	Username       string   `json:"username"`
	Password       string   `json:"password,omitempty"` // sealed; stripped from responses
	Folder         string   `json:"folder"`
	FromFilter     []string `json:"from_filter"` // addresses or @domains; empty = any sender
	SubjectFilter  string   `json:"subject_filter"`
	SinceDate      string   `json:"since_date"` // YYYY-MM-DD; mail before this is ignored
	Mode           string   `json:"mode"`
	DepartmentID   int64    `json:"department_id"`
	TicketCategory string   `json:"ticket_category"`
	KBCategory     string   `json:"kb_category"`
	AIEnabled      bool     `json:"ai_enabled"` // knowledge from this mailbox may be used by the AI
	SMTPHost       string   `json:"smtp_host"`  // optional: send ticket replies to customers
	SMTPPort       int      `json:"smtp_port"`
	SMTPSecurity   string   `json:"smtp_security"` // tls, starttls or none
	FromAddress    string   `json:"from_address"`
	PollMinutes    int      `json:"poll_minutes"`
	Enabled        bool     `json:"enabled"`
	ConsentBy      int64    `json:"consent_by"`
	ConsentAt      string   `json:"consent_at"`
	Status         string   `json:"status"` // ok, error, never
	LastSync       string   `json:"last_sync"`
	LastError      string   `json:"last_error"`
	UIDValidity    uint32   `json:"uid_validity"`
	LastUID        uint32   `json:"last_uid"`
	Imported       int      `json:"imported"`
	HasPassword    bool     `json:"has_password"`
}

// EmailMessage is an imported message (body encrypted at rest).
type EmailMessage struct {
	ID          int64     `json:"id"`
	TenantID    int64     `json:"tenant_id"`
	ConnectorID int64     `json:"connector_id"`
	UID         uint32    `json:"uid"`
	MessageID   string    `json:"message_id"`
	InReplyTo   []string  `json:"in_reply_to"`
	FromAddr    string    `json:"from_addr"`
	FromName    string    `json:"from_name"`
	Subject     string    `json:"subject"`
	Date        time.Time `json:"date"`
	Body        string    `json:"body,omitempty"`
	TicketID    int64     `json:"ticket_id"`
	KBSourceID  int64     `json:"kb_source_id"`
	CreatedAt   time.Time `json:"created_at"`
}
