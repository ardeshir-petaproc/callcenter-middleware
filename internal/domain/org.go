package domain

import "time"

// Tenant is one business using the platform; all its data is isolated.
type Tenant struct {
	ID        int64     `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Status    string    `json:"status"` // active, suspended
	Plan      string    `json:"plan"`
	Settings  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

// Tenant statuses.
const (
	TenantActive    = "active"
	TenantSuspended = "suspended"
)

// Department groups teams (queues) and employees, e.g. sales, support.
type Department struct {
	ID          int64    `json:"id"`
	TenantID    int64    `json:"tenant_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	ManagerID   int64    `json:"manager_id"`
	ScheduleID  int64    `json:"schedule_id"`
	Capacity    int      `json:"capacity"` // max concurrent calls (0 = unlimited)
	Priority    int      `json:"priority"`
	Fallback    Action   `json:"fallback"` // used when closed, over capacity or unstaffed
	Languages   []string `json:"languages"`
	Enabled     bool     `json:"enabled"`
}

// JobProfile is a job description: responsibilities, the request types the
// holder may handle, authority limits and escalation conditions.
type JobProfile struct {
	ID                   int64    `json:"id"`
	TenantID             int64    `json:"tenant_id"`
	Title                string   `json:"title"`
	DepartmentID         int64    `json:"department_id"`
	Description          string   `json:"description"`
	Responsibilities     []string `json:"responsibilities"`
	RequestTypes         []string `json:"request_types"` // topic keys
	AuthorityLimits      string   `json:"authority_limits"`
	EscalationConditions string   `json:"escalation_conditions"`
	EscalateToUserID     int64    `json:"escalate_to_user_id"`
	IsTemplate           bool     `json:"is_template"`
	Version              int      `json:"version"`
	UpdatedAt            string   `json:"updated_at"`
}

// PhoneNumber maps a dialed number (DNIS) to a tenant and entry flow.
type PhoneNumber struct {
	ID       int64  `json:"id"`
	TenantID int64  `json:"tenant_id"`
	Number   string `json:"number"`
	Label    string `json:"label"`
	FlowID   int64  `json:"flow_id"`
	Enabled  bool   `json:"enabled"`
}

// CustomRole is a tenant-defined set of permissions.
type CustomRole struct {
	ID          int64    `json:"id"`
	TenantID    int64    `json:"tenant_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// Invitation brings a person into a tenant, optionally linked to a Joow account.
type Invitation struct {
	ID            int64   `json:"id"`
	TenantID      int64   `json:"tenant_id"`
	Name          string  `json:"name"`
	Email         string  `json:"email"`
	Phone         string  `json:"phone"`
	JoowAccountID string  `json:"joow_account_id"`
	Role          Role    `json:"role"`
	CustomRoleID  int64   `json:"custom_role_id"`
	DepartmentIDs []int64 `json:"department_ids"`
	TokenHash     string  `json:"token_hash,omitempty"` // stored; stripped from API responses
	Status        string  `json:"status"`               // pending, accepted, revoked
	ExpiresAt     string  `json:"expires_at"`
	InvitedBy     int64   `json:"invited_by"`
	UserID        int64   `json:"user_id"`
}

// Invitation statuses.
const (
	InvitePending  = "pending"
	InviteAccepted = "accepted"
	InviteRevoked  = "revoked"
)

// Revision is a historical version of a configuration object.
type Revision struct {
	ID            int64     `json:"id"`
	TenantID      int64     `json:"tenant_id"`
	Entity        string    `json:"entity"`
	EntityID      int64     `json:"entity_id"`
	Action        string    `json:"action"`
	Data          string    `json:"data"`
	ChangedBy     int64     `json:"changed_by"`
	ChangedByName string    `json:"changed_by_name"`
	At            time.Time `json:"at"`
}
