package domain

import "time"

// TelephonyEventType categorizes normalized events from Asterisk ARI.
type TelephonyEventType string

const (
	EventCallStarted       TelephonyEventType = "CALL_STARTED"
	EventPlaybackFinished  TelephonyEventType = "PLAYBACK_FINISHED"
	EventDTMFReceived      TelephonyEventType = "DTMF_RECEIVED"
	EventCallEnded         TelephonyEventType = "CALL_ENDED"
	EventChannelDestroyed  TelephonyEventType = "CHANNEL_DESTROYED"
	EventRecordingFinished TelephonyEventType = "RECORDING_FINISHED"
	EventRecordingFailed   TelephonyEventType = "RECORDING_FAILED"
	EventStreamConnected   TelephonyEventType = "STREAM_CONNECTED"
)

// TelephonyEvent provides decoupled telephony facts for the call core.
type TelephonyEvent struct {
	Type          TelephonyEventType
	ChannelID     string
	ChannelName   string
	CallerNumber  string
	CallerName    string
	DNIS          string
	Args          []string
	PlaybackID    string
	Digit         string
	Cause         int
	CauseText     string
	RecordingName string
	DurationSec   int
	Timestamp     time.Time
}

// HandoffNotification holds the payload sent to screen-pop receivers.
type HandoffNotification struct {
	ChannelID    string    `json:"channel_id"`
	CallID       string    `json:"call_id"`
	CallerNumber string    `json:"caller_number"`
	CallerName   string    `json:"caller_name"`
	TargetQueue  string    `json:"target_queue"`
	InputDigits  string    `json:"input_digits"`
	Reason       string    `json:"reason"`
	Timestamp    time.Time `json:"timestamp"`
}

// Event is a realtime notification published to dashboards and webhooks.
type Event struct {
	Type string    `json:"type"`
	Data any       `json:"data"`
	At   time.Time `json:"at"`
}

// Realtime event types.
const (
	EvCallUpdated     = "call.updated"
	EvCallEnded       = "call.ended"
	EvAgentUpdated    = "agent.updated"
	EvQueueUpdated    = "queue.updated"
	EvTranscript      = "call.transcript"
	EvCallbackCreated = "callback.created"
	EvVoicemail       = "voicemail.created"
	EvScreenPop       = "agent.screenpop"
	EvConfigChanged   = "config.changed"
	EvQueueAlert      = "queue.alert"
)
