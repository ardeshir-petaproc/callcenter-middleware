package domain

import "time"

// TelephonyEventType categorizes normalized events from Asterisk ARI.
type TelephonyEventType string

const (
	EventCallStarted       TelephonyEventType = "CALL_STARTED"
	EventPlaybackFinished  TelephonyEventType = "PLAYBACK_FINISHED"
	EventDTMFReceived      TelephonyEventType = "DTMF_RECEIVED"
	EventCallEnded         TelephonyEventType = "CALL_ENDED"
)

// TelephonyEvent provides decoupled telephony facts for the IVR engine.
type TelephonyEvent struct {
	Type         TelephonyEventType
	ChannelID    string
	CallerNumber string
	CallerName   string
	PlaybackID   string
	Digit        string
	Timestamp    time.Time
}

// HandoffNotification holds payload sent to screen-pop receivers.
type HandoffNotification struct {
	ChannelID    string    `json:"channel_id"`
	CallerNumber string    `json:"caller_number"`
	CallerName   string    `json:"caller_name"`
	TargetQueue  string    `json:"target_queue"`
	InputDigits  string    `json:"input_digits"`
	Reason       string    `json:"reason"`
	Timestamp    time.Time `json:"timestamp"`
}
