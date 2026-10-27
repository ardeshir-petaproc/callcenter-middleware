package domain

import "context"

// OriginateRequest describes a new outbound channel placed into Stasis.
type OriginateRequest struct {
	Endpoint   string
	ChannelID  string
	CallerID   string
	TimeoutSec int
	AppArgs    string
	Variables  map[string]string
}

// ExternalMediaRequest describes an ExternalMedia (RTP) channel.
type ExternalMediaRequest struct {
	ChannelID    string
	ExternalHost string
	Format       string
}

// SnoopRequest describes a snoop channel for supervisor monitoring.
type SnoopRequest struct {
	ChannelID string
	SnoopID   string
	Spy       string
	Whisper   string
	AppArgs   string
}

// SwitchController abstracts Asterisk ARI REST commands.
type SwitchController interface {
	AnswerChannel(ctx context.Context, channelID string) error
	PlayMedia(ctx context.Context, channelID, mediaURI, playbackID string) error
	StopPlayback(ctx context.Context, playbackID string) error
	ContinueToQueue(ctx context.Context, channelID, contextName, queueExt string, priority int) error
	HangupChannel(ctx context.Context, channelID, reason string) error
	Originate(ctx context.Context, req OriginateRequest) error
	CreateBridge(ctx context.Context, bridgeID, bridgeType string) error
	DestroyBridge(ctx context.Context, bridgeID string) error
	AddToBridge(ctx context.Context, bridgeID string, channelIDs ...string) error
	RemoveFromBridge(ctx context.Context, bridgeID, channelID string) error
	StartMOH(ctx context.Context, channelID, class string) error
	StopMOH(ctx context.Context, channelID string) error
	StartRinging(ctx context.Context, channelID string) error
	StopRinging(ctx context.Context, channelID string) error
	RecordBridge(ctx context.Context, bridgeID, name string) error
	RecordChannel(ctx context.Context, channelID, name string, maxSec int, beep bool) error
	StopRecording(ctx context.Context, name string) error
	// ExternalMedia returns the "ip:port" where Asterisk receives RTP (may be empty).
	ExternalMedia(ctx context.Context, req ExternalMediaRequest) (string, error)
	Snoop(ctx context.Context, req SnoopRequest) error
	SetMute(ctx context.Context, channelID string, mute bool) error
	ListChannels(ctx context.Context) ([]string, error)
}

// EventSource streams normalized telephony events from Asterisk ARI WebSocket.
type EventSource interface {
	Subscribe(ctx context.Context, out chan<- TelephonyEvent) error
	IsConnected() bool
}

// ScreenPopNotifier pushes incoming call context to office employees.
type ScreenPopNotifier interface {
	NotifyQueueHandoff(ctx context.Context, payload HandoffNotification) error
}

// Publisher fans realtime events out to dashboards and webhooks.
type Publisher interface {
	Publish(ev Event)
}

// MetricsRecorder decouples domain business logic from Prometheus specifics.
type MetricsRecorder interface {
	SetWSConnected(connected bool)
	IncActiveCalls()
	DecActiveCalls()
	RecordDTMF(digit string, outcome string)
	RecordARILatency(operation string, durationSeconds float64)
	RecordHandoff(queueExt string, reason string)
	RecordOutcome(outcome string)
	ObserveWait(queue string, seconds float64)
	SetQueueWaiting(queue string, n int)
	SetAgentsByStatus(counts map[string]int)
	SetAISessions(n int)
}
