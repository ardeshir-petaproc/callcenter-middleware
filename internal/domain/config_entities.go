package domain

// ActionType enumerates what can happen to a caller at a decision point.
type ActionType string

const (
	ActionNone      ActionType = ""
	ActionQueue     ActionType = "queue"
	ActionAI        ActionType = "ai"
	ActionVoicemail ActionType = "voicemail"
	ActionCallback  ActionType = "callback"
	ActionHangup    ActionType = "hangup"
	ActionFlow      ActionType = "flow"
	ActionAgent     ActionType = "agent"
	ActionExternal  ActionType = "external"
	ActionDialplan  ActionType = "dialplan"
)

// Action is a routing instruction. Target holds an id, number or extension
// depending on Type (queue id, AI profile id, flow id, user id, phone number).
type Action struct {
	Type   ActionType `json:"type"`
	Target string     `json:"target,omitempty"`
	Media  string     `json:"media,omitempty"`
}

// Queue strategies supported by the native ACD.
const (
	StrategyRingAll     = "ringall"
	StrategyLongestIdle = "longest_idle"
	StrategyRoundRobin  = "round_robin"
	StrategyFewestCalls = "fewest_calls"
	StrategyRandom      = "random"
	StrategyLinear      = "linear" // sequential ring in member order
)

// Queue modes: native routes agents through the middleware ACD, asterisk
// hands the caller to an app_queue extension in the dialplan.
const (
	QueueModeNative   = "native"
	QueueModeAsterisk = "asterisk"
)

// Queue describes an ACD queue.
type Queue struct {
	ID             int64    `json:"id"`
	TenantID       int64    `json:"tenant_id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Mode           string   `json:"mode"`
	Extension      string   `json:"extension"`
	Strategy       string   `json:"strategy"`
	Priority       int      `json:"priority"`
	RingTimeoutSec int      `json:"ring_timeout_sec"`
	MaxWaitSec     int      `json:"max_wait_sec"`
	WrapupSec      int      `json:"wrapup_sec"`
	SLASec         int      `json:"sla_sec"`
	MOHClass       string   `json:"moh_class"`
	JoinMedia      string   `json:"join_media"`
	RequiredSkills []string `json:"required_skills"`
	Record         bool     `json:"record"`
	RecordNotice   string   `json:"record_notice"` // played before queueing when recording (consent notice)
	AllowCallback  bool     `json:"allow_callback"`
	LeaveWhenEmpty bool     `json:"leave_when_empty"`
	Survey         bool     `json:"survey"`
	DepartmentID   int64    `json:"department_id"`
	AlertWaiting   int      `json:"alert_waiting"`
	AlertWaitSec   int      `json:"alert_wait_sec"`
	Overflow       Action   `json:"overflow"`
	Enabled        bool     `json:"enabled"`
}

// QueueMember assigns a user to a queue with a penalty (lower rings first).
type QueueMember struct {
	TenantID int64 `json:"tenant_id"`
	QueueID  int64 `json:"queue_id"`
	UserID   int64 `json:"user_id"`
	Penalty  int   `json:"penalty"`
}

// IVRFlow is a data-driven DTMF menu bound to a dialed number (DNIS).
type IVRFlow struct {
	ID               int64             `json:"id"`
	TenantID         int64             `json:"tenant_id"`
	Name             string            `json:"name"`
	DNIS             string            `json:"dnis"`
	GreetingMedia    string            `json:"greeting_media"`
	InvalidMedia     string            `json:"invalid_media"`
	TimeoutSec       int               `json:"timeout_sec"`
	MaxRetries       int               `json:"max_retries"`
	Options          map[string]Action `json:"options"`
	TimeoutAction    Action            `json:"timeout_action"`
	DirectAction     Action            `json:"direct_action"`
	ScheduleID       int64             `json:"schedule_id"`
	AfterHoursAction Action            `json:"after_hours_action"`
	Enabled          bool              `json:"enabled"`
}

// TimeRange is an opening window on a weekday (0=Sunday) in HH:MM.
type TimeRange struct {
	Day   int    `json:"day"`
	Start string `json:"start"`
	End   string `json:"end"`
}

// Holiday closes a schedule for a whole date (YYYY-MM-DD).
type Holiday struct {
	Date string `json:"date"`
	Name string `json:"name"`
}

// Schedule defines business hours.
type Schedule struct {
	ID       int64       `json:"id"`
	TenantID int64       `json:"tenant_id"`
	Name     string      `json:"name"`
	Timezone string      `json:"timezone"`
	Weekly   []TimeRange `json:"weekly"`
	Holidays []Holiday   `json:"holidays"`
}

// AI tools that a profile can enable.
const (
	ToolTransferToQueue = "transfer_to_queue"
	ToolEndCall         = "end_call"
	ToolCreateCallback  = "create_callback"
	ToolLookupContact   = "lookup_contact"
	ToolTakeMessage     = "take_message"
	ToolCheckHours      = "check_business_hours"
)

// Disposition is a wrap-up code an agent selects after a call.
type Disposition struct {
	ID       int64  `json:"id"`
	TenantID int64  `json:"tenant_id"`
	Code     string `json:"code"`
	Label    string `json:"label"`
	Enabled  bool   `json:"enabled"`
}

// PauseReason is a selectable reason for an agent pause.
type PauseReason struct {
	ID       int64  `json:"id"`
	TenantID int64  `json:"tenant_id"`
	Label    string `json:"label"`
	Enabled  bool   `json:"enabled"`
}

// Webhook receives signed event notifications.
type Webhook struct {
	ID        int64    `json:"id"`
	TenantID  int64    `json:"tenant_id"`
	URL       string   `json:"url"`
	Events    []string `json:"events"`
	Secret    string   `json:"secret"` // sealed at rest; never returned
	Enabled   bool     `json:"enabled"`
	HasSecret bool     `json:"has_secret"`
}

// Contact is a lightweight CRM record used for screen-pops.
type Contact struct {
	ID       int64    `json:"id"`
	TenantID int64    `json:"tenant_id"`
	Name     string   `json:"name"`
	Phone    string   `json:"phone"`
	Email    string   `json:"email"`
	Company  string   `json:"company"`
	Notes    string   `json:"notes"`
	Tags     []string `json:"tags"`
	VIP      bool     `json:"vip"`
}
