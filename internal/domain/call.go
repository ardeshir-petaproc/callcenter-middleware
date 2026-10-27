package domain

import (
	"context"
	"time"
)

// CallState tracks the discrete stages of an inbound call.
type CallState string

const (
	StateNew        CallState = "NEW"
	StateGreeting   CallState = "GREETING"
	StateRouting    CallState = "ROUTING"
	StateAIAgent    CallState = "AI_AGENT"
	StateHandoff    CallState = "HANDOFF"
	StateTerminated CallState = "TERMINATED"
)

// CallSession represents an active Stasis channel session.
type CallSession struct {
	ChannelID     string
	CallerNumber  string
	CallerName    string
	CurrentState  CallState
	ActivePlayID  string
	InputDigits   string
	RetryCount    int
	StartTime     time.Time
	CancelTimer   context.CancelFunc
}
