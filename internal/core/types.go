package core

import (
	"time"

	"callcenter-middleware/internal/domain"
)

// Leg roles identify what a channel is doing for a call.
const (
	roleCaller     = "caller"
	roleAgent      = "agent"
	roleMedia      = "media"
	roleSnoop      = "snoop"
	roleSupervisor = "supervisor"
	roleOutAgent   = "outagent"
	roleOutCust    = "outcust"
	roleExternal   = "external"
	roleConsult    = "consult" // warm-transfer / conference target
)

// leg is one Asterisk channel participating in a call.
type leg struct {
	chanID   string
	call     *liveCall
	role     string
	agentID  int64
	answered bool
	since    time.Time
	mode     string // supervisor monitor mode
	pairChan string // supervisor <-> snoop pairing
	bridgeID string // monitor bridge
}

// liveCall is the in-memory state of a call controlled by the core.
type liveCall struct {
	domain.Call

	callerChan string
	agentChan  string
	bridgeID   string
	mediaChan  string
	extChan    string

	ringing map[string]int64 // agent channel -> user id
	tried   map[int64]bool

	flow          *domain.IVRFlow
	retries       int
	playID        string
	afterPlay     func()
	inputDeadline time.Time

	queue        *domain.Queue
	queueEnter   time.Time
	priority     int
	targetAgent  int64
	holdStart    time.Time
	aiStart      time.Time
	aiHandoff    int64
	callbackID   int64
	leadID       int64
	surveyUntil  time.Time
	talkEnd      time.Time
	transferring bool
	consultChan  string // warm-transfer target leg while the caller is on hold
	conference   bool   // caller, agent and consult target are all bridged
	returnTo     int64  // agent to return to if a blind transfer to an agent fails
	announcing   bool   // recording notice playing; agents are not rung yet
	ended        bool
}

func newLiveCall(id string) *liveCall {
	return &liveCall{
		Call:    domain.Call{ID: id, Tags: []string{}},
		ringing: map[string]int64{},
		tried:   map[int64]bool{},
	}
}

// agentRT is the runtime state of an agent.
type agentRT struct {
	domain.AgentState
	user          domain.User
	pendingStatus domain.AgentStatus
	pendingReason string
	idleSince     time.Time
	retryAfter    time.Time // backoff after a failed origination
}

// recordingRef remembers who a voicemail recording belongs to.
type recordingRef struct {
	tenant  int64
	callID  string
	number  string
	name    string
	queueID int64
	beep    bool
	started time.Time
}

// CallView is the live representation of a call returned to the API.
type CallView struct {
	domain.Call
	RingingAgents []int64 `json:"ringing_agents"`
	WaitingSec    int     `json:"waiting_sec"`
	DurationSec   int     `json:"duration_sec"`
	Monitors      int     `json:"monitors"`
	Consult       string  `json:"consult,omitempty"` // ringing, talking or conference
	ConsultAgent  int64   `json:"consult_agent_id,omitempty"`
}

// QueueView is the live status of one queue.
type QueueView struct {
	ID             int64  `json:"id"`
	Name           string `json:"name"`
	Waiting        int    `json:"waiting"`
	LongestWaitSec int    `json:"longest_wait_sec"`
	Available      int    `json:"available"`
	Busy           int    `json:"busy"`
	Paused         int    `json:"paused"`
	LoggedIn       int    `json:"logged_in"`
	Members        int    `json:"members"`
}

// Snapshot is a consistent view of all live state.
type Snapshot struct {
	Calls  []CallView          `json:"calls"`
	Agents []domain.AgentState `json:"agents"`
	Queues []QueueView         `json:"queues"`
	AI     int                 `json:"ai_sessions"`
	Alerts []QueueAlert        `json:"alerts"`
	At     time.Time           `json:"at"`
}

// Supervisor monitor modes.
const (
	MonitorSpy     = "spy"
	MonitorWhisper = "whisper"
	MonitorBarge   = "barge"
)
