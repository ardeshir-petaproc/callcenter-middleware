package domain

import "time"

// Role controls what a user may do in the control panel and API.
type Role string

const (
	RoleAdmin      Role = "admin"
	RoleSupervisor Role = "supervisor"
	RoleAgent      Role = "agent"
)

// Rank orders roles so that checks can be expressed as "at least supervisor".
func (r Role) Rank() int {
	switch r {
	case RoleAdmin:
		return 3
	case RoleSupervisor:
		return 2
	case RoleAgent:
		return 1
	}
	return 0
}

// Valid reports whether r is a known role.
func (r Role) Valid() bool { return r.Rank() > 0 }

// User is an operator of the call center: admin, supervisor or agent.
type User struct {
	ID           int64     `json:"id"`
	TenantID     int64     `json:"tenant_id"`
	Username     string    `json:"username"`
	DisplayName  string    `json:"display_name"`
	Email        string    `json:"email"`
	Role         Role      `json:"role"`
	Extension    string    `json:"extension"`
	Endpoint     string    `json:"endpoint"`
	Skills       []string  `json:"skills"`
	Enabled      bool      `json:"enabled"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	// Identity and access.
	JoowAccountID string `json:"joow_account_id"`
	CustomRoleID  int64  `json:"custom_role_id"`
	PlatformAdmin bool   `json:"platform_admin"`
	Status        string `json:"status"` // active, invited, disabled

	Profile
}

// Profile is the organizational profile of an employee. It is also an
// input of the routing engine (departments, languages, job profile).
type Profile struct {
	Title         string   `json:"title"`
	DepartmentIDs []int64  `json:"department_ids"`
	ManagerID     int64    `json:"manager_id"`
	Languages     []string `json:"languages"`
	ScheduleID    int64    `json:"schedule_id"`
	JobProfileID  int64    `json:"job_profile_id"`
	Phone         string   `json:"phone"`
}

// User statuses.
const (
	UserActive   = "active"
	UserInvited  = "invited"
	UserDisabled = "disabled"
)

// InDepartment reports whether the user belongs to department id.
func (u *User) InDepartment(id int64) bool {
	for _, d := range u.DepartmentIDs {
		if d == id {
			return true
		}
	}
	return false
}

// SpeaksAny reports whether the user speaks one of langs (or langs is empty).
func (u *User) SpeaksAny(langs []string) bool {
	if len(langs) == 0 {
		return true
	}
	for _, want := range langs {
		for _, l := range u.Languages {
			if l == want {
				return true
			}
		}
	}
	return false
}

// DialEndpoint returns the Asterisk endpoint used to ring this user.
func (u *User) DialEndpoint() string {
	if u.Endpoint != "" {
		return u.Endpoint
	}
	if u.Extension != "" {
		return "PJSIP/" + u.Extension
	}
	return ""
}

// HasSkills reports whether the user has every skill in required.
func (u *User) HasSkills(required []string) bool {
	for _, req := range required {
		found := false
		for _, s := range u.Skills {
			if s == req {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// AgentStatus is the realtime presence of an agent.
type AgentStatus string

const (
	AgentOffline   AgentStatus = "offline"
	AgentAvailable AgentStatus = "available"
	AgentRinging   AgentStatus = "ringing"
	AgentBusy      AgentStatus = "busy"
	AgentWrapup    AgentStatus = "wrapup"
	AgentPaused    AgentStatus = "paused"
)

// AgentState is the live view of an agent maintained by the call core.
type AgentState struct {
	UserID      int64       `json:"user_id"`
	TenantID    int64       `json:"tenant_id"`
	Username    string      `json:"username"`
	DisplayName string      `json:"display_name"`
	Extension   string      `json:"extension"`
	Status      AgentStatus `json:"status"`
	PauseReason string      `json:"pause_reason,omitempty"`
	Since       time.Time   `json:"since"`
	CallID      string      `json:"call_id,omitempty"`
	LastCallEnd time.Time   `json:"last_call_end,omitzero"`
	CallsToday  int         `json:"calls_today"`
	TalkSecs    int         `json:"talk_secs_today"`
	MissedInRow int         `json:"missed_in_row"`
	WrapupUntil time.Time   `json:"wrapup_until,omitzero"`
	QueueIDs    []int64     `json:"queue_ids"`
}

// AgentEvent is a persisted status transition used for reporting.
type AgentEvent struct {
	TenantID int64       `json:"tenant_id"`
	UserID   int64       `json:"user_id"`
	Status   AgentStatus `json:"status"`
	Reason   string      `json:"reason"`
	At       time.Time   `json:"at"`
}
