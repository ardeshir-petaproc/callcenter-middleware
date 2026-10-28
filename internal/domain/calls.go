package domain

import "time"

// CallState is the live phase of a call.
type CallState string

const (
	CallIVR        CallState = "ivr"
	CallAI         CallState = "ai"
	CallQueued     CallState = "queued"
	CallRinging    CallState = "ringing"
	CallConnected  CallState = "connected"
	CallHold       CallState = "hold"
	CallVoicemail  CallState = "voicemail"
	CallDialing    CallState = "dialing"
	CallMessage    CallState = "message"
	CallSurvey     CallState = "survey"
	CallEnded      CallState = "ended"
	CallTransfered CallState = "transferred_out"
)

// CallOutcome is the final result recorded when a call ends.
type CallOutcome string

const (
	OutcomeAnswered    CallOutcome = "answered"
	OutcomeAbandoned   CallOutcome = "abandoned"
	OutcomeVoicemail   CallOutcome = "voicemail"
	OutcomeCallback    CallOutcome = "callback_requested"
	OutcomeAIResolved  CallOutcome = "ai_resolved"
	OutcomeDialplan    CallOutcome = "sent_to_dialplan"
	OutcomeFailed      CallOutcome = "failed"
	OutcomeNoAnswer    CallOutcome = "no_answer"
	OutcomeHangupIVR   CallOutcome = "ivr_hangup"
	OutcomeRejected    CallOutcome = "rejected"
	OutcomeTransferred CallOutcome = "transferred_external"
)

// Call directions.
const (
	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"
)

// Call is a call detail record plus live state.
type Call struct {
	ID           string      `json:"id"`
	TenantID     int64       `json:"tenant_id"`
	Direction    string      `json:"direction"`
	CallerNumber string      `json:"caller_number"`
	CallerName   string      `json:"caller_name"`
	DNIS         string      `json:"dnis"`
	State        CallState   `json:"state"`
	Outcome      CallOutcome `json:"outcome"`
	FlowID       int64       `json:"flow_id"`
	QueueID      int64       `json:"queue_id"`
	AgentID      int64       `json:"agent_id"`
	AIProfileID  int64       `json:"ai_profile_id"`
	ContactID    int64       `json:"contact_id"`
	IVRPath      string      `json:"ivr_path"`
	StartedAt    time.Time   `json:"started_at"`
	QueuedAt     *time.Time  `json:"queued_at,omitempty"`
	AnsweredAt   *time.Time  `json:"answered_at,omitempty"`
	EndedAt      *time.Time  `json:"ended_at,omitempty"`
	WaitSec      int         `json:"wait_sec"`
	TalkSec      int         `json:"talk_sec"`
	HoldSec      int         `json:"hold_sec"`
	SLAMet       bool        `json:"sla_met"`
	Disposition  string      `json:"disposition"`
	Notes        string      `json:"notes"`
	Tags         []string    `json:"tags"`
	Recording    string      `json:"recording"`
	Summary      string      `json:"summary"`
	Sentiment    string      `json:"sentiment"`
	HangupBy     string      `json:"hangup_by"`
	CSAT         int         `json:"csat"`
	CampaignID   int64       `json:"campaign_id"`
}

// TranscriptEntry is one utterance in a call.
type TranscriptEntry struct {
	CallID  string    `json:"call_id"`
	Speaker string    `json:"speaker"`
	Text    string    `json:"text"`
	At      time.Time `json:"at"`
}

// CallEvent is one entry in a call timeline.
type CallEvent struct {
	CallID string    `json:"call_id"`
	Type   string    `json:"type"`
	Detail string    `json:"detail"`
	At     time.Time `json:"at"`
}

// CallFilter narrows call history queries.
type CallFilter struct {
	From       time.Time
	To         time.Time
	QueueID    int64
	AgentID    int64
	Outcome    string
	Direction  string
	Search     string
	CampaignID int64
	Limit      int
	Offset     int
}

// Callback is a request to call a customer back.
type Callback struct {
	ID         int64      `json:"id"`
	TenantID   int64      `json:"tenant_id"`
	CallID     string     `json:"call_id"`
	Number     string     `json:"number"`
	Name       string     `json:"name"`
	QueueID    int64      `json:"queue_id"`
	Status     string     `json:"status"`
	AssignedTo int64      `json:"assigned_to"`
	DueAt      *time.Time `json:"due_at,omitempty"`
	Attempts   int        `json:"attempts"`
	Notes      string     `json:"notes"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// Callback statuses.
const (
	CallbackPending    = "pending"
	CallbackInProgress = "in_progress"
	CallbackDone       = "done"
	CallbackFailed     = "failed"
	CallbackCancelled  = "cancelled"
)

// Voicemail is a recorded caller message.
type Voicemail struct {
	ID           int64     `json:"id"`
	TenantID     int64     `json:"tenant_id"`
	CallID       string    `json:"call_id"`
	CallerNumber string    `json:"caller_number"`
	CallerName   string    `json:"caller_name"`
	QueueID      int64     `json:"queue_id"`
	Recording    string    `json:"recording"`
	DurationSec  int       `json:"duration_sec"`
	Transcript   string    `json:"transcript"`
	Heard        bool      `json:"heard"`
	CreatedAt    time.Time `json:"created_at"`
}

// AuditEntry records an administrative action.
type AuditEntry struct {
	ID       int64     `json:"id"`
	TenantID int64     `json:"tenant_id"`
	UserID   int64     `json:"user_id"`
	Username string    `json:"username"`
	Action   string    `json:"action"`
	Entity   string    `json:"entity"`
	EntityID string    `json:"entity_id"`
	Detail   string    `json:"detail"`
	At       time.Time `json:"at"`
}
