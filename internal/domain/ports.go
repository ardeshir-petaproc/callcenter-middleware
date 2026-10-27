package domain

import "context"

// SwitchController abstracts Asterisk ARI REST commands.
type SwitchController interface {
	AnswerChannel(ctx context.Context, channelID string) error
	PlayMedia(ctx context.Context, channelID, mediaURI, playbackID string) error
	StopPlayback(ctx context.Context, playbackID string) error
	ContinueToQueue(ctx context.Context, channelID, contextName, queueExt string, priority int) error
	HangupChannel(ctx context.Context, channelID, reason string) error
}

// EventSource streams normalized telephony events from Asterisk ARI WebSocket.
type EventSource interface {
	Subscribe(ctx context.Context, out chan<- TelephonyEvent) error
	IsConnected() bool
}

// CallStateHandler represents a modular step in the call lifecycle.
type CallStateHandler interface {
	Name() CallState
	OnEnter(ctx context.Context, session *CallSession) error
	OnEvent(ctx context.Context, session *CallSession, ev TelephonyEvent) (nextState CallState, err error)
}

// ScreenPopNotifier pushes incoming call context to office employees.
type ScreenPopNotifier interface {
	NotifyQueueHandoff(ctx context.Context, payload HandoffNotification) error
}

// VoiceAgentBridge is the Phase 2 hook for external AI voice streaming.
type VoiceAgentBridge interface {
	StartVoiceSession(ctx context.Context, session *CallSession) error
	StopVoiceSession(ctx context.Context, channelID string) error
}

// MetricsRecorder decouples domain business logic from Prometheus specifics.
type MetricsRecorder interface {
	SetWSConnected(connected bool)
	IncActiveCalls()
	DecActiveCalls()
	RecordDTMF(digit string, outcome string)
	RecordARILatency(operation string, durationSeconds float64)
	RecordHandoff(queueExt string, reason string)
}
