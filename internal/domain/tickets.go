package domain

import "time"

// Ticket statuses.
const (
	TicketNew        = "new"
	TicketAssigned   = "assigned"
	TicketInProgress = "in_progress"
	TicketWaiting    = "waiting_customer"
	TicketResolved   = "resolved"
	TicketClosed     = "closed"
)

// TicketStatuses lists the statuses in workflow order.
var TicketStatuses = []string{TicketNew, TicketAssigned, TicketInProgress, TicketWaiting, TicketResolved, TicketClosed}

// Ticket priorities.
const (
	PriorityLow    = "low"
	PriorityNormal = "normal"
	PriorityHigh   = "high"
	PriorityUrgent = "urgent"
)

// Ticket is a customer request tracked until resolution.
type Ticket struct {
	ID             int64      `json:"id"`
	TenantID       int64      `json:"tenant_id"`
	Number         string     `json:"number"`
	Subject        string     `json:"subject"`
	Description    string     `json:"description"`
	Category       string     `json:"category"`
	Priority       string     `json:"priority"`
	Status         string     `json:"status"`
	Source         string     `json:"source"` // call, email, manual, ai
	DepartmentID   int64      `json:"department_id"`
	AssigneeID     int64      `json:"assignee_id"`
	ContactID      int64      `json:"contact_id"`
	RequesterName  string     `json:"requester_name"`
	RequesterPhone string     `json:"requester_phone"`
	RequesterEmail string     `json:"requester_email"`
	CallIDs        []string   `json:"call_ids"`
	Tags           []string   `json:"tags"`
	CreatedBy      int64      `json:"created_by"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	FirstResponse  *time.Time `json:"first_response_at,omitempty"`
	ResolvedAt     *time.Time `json:"resolved_at,omitempty"`
	ClosedAt       *time.Time `json:"closed_at,omitempty"`
	ResponseDue    *time.Time `json:"response_due_at,omitempty"`
	ResolutionDue  *time.Time `json:"resolution_due_at,omitempty"`
}

// Open reports whether the ticket still needs work.
func (t *Ticket) Open() bool { return t.Status != TicketResolved && t.Status != TicketClosed }

// Overdue reports whether an SLA deadline has passed while open.
func (t *Ticket) Overdue(now time.Time) bool {
	if !t.Open() {
		return false
	}
	if t.ResolutionDue != nil && now.After(*t.ResolutionDue) {
		return true
	}
	return t.FirstResponse == nil && t.ResponseDue != nil && now.After(*t.ResponseDue)
}

// TicketComment is a reply visible to the customer or an internal note.
type TicketComment struct {
	ID         int64     `json:"id"`
	TicketID   int64     `json:"ticket_id"`
	AuthorID   int64     `json:"author_id"`
	AuthorName string    `json:"author_name"`
	Body       string    `json:"body"`
	Internal   bool      `json:"internal"`
	CreatedAt  time.Time `json:"created_at"`
}

// TicketEvent is one entry in a ticket's change history.
type TicketEvent struct {
	ID        int64     `json:"id"`
	TicketID  int64     `json:"ticket_id"`
	Type      string    `json:"type"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	ActorID   int64     `json:"actor_id"`
	ActorName string    `json:"actor_name"`
	At        time.Time `json:"at"`
}

// SLAPolicy sets response and resolution targets per priority.
type SLAPolicy struct {
	ID                   int64  `json:"id"`
	TenantID             int64  `json:"tenant_id"`
	Priority             string `json:"priority"`
	FirstResponseMinutes int    `json:"first_response_minutes"`
	ResolutionMinutes    int    `json:"resolution_minutes"`
}

// TicketFilter narrows ticket queries.
type TicketFilter struct {
	Status       string
	Priority     string
	Category     string
	DepartmentID int64
	AssigneeID   int64
	Search       string
	CallID       string
	OverdueAt    *time.Time
	VisibleTo    *TicketVisibility
	Limit        int
	Offset       int
}

// TicketVisibility restricts tickets to those a user works on.
type TicketVisibility struct {
	UserID        int64
	DepartmentIDs []int64
}
