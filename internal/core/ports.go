package core

import (
	"callcenter-middleware/internal/config"
	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
	"context"
)

// AISpec describes an AI voice session to start.
type AISpec struct {
	CallID       string
	CallerNumber string
	CallerName   string
	Profile      domain.AIProfile
}

// AIControl lets AI sessions act on their call.
type AIControl interface {
	AITransfer(callID, queue, reason, summary string)
	AIEndCall(callID, summary string)
	AICallback(callID, number, notes string)
	AITranscript(callID, speaker, text string)
	AIFailed(callID string, err error)
}

// AIManager runs AI voice sessions bridged through ExternalMedia.
type AIManager interface {
	Start(spec AISpec, ctl AIControl) (externalHost string, err error)
	Stop(callID string)
	Active() int
}

// Analyzer performs post-call analysis (summary, sentiment) asynchronously.
type Analyzer interface {
	Analyze(ctx context.Context, callID string)
}

// Deps bundles the collaborators of the core.
type Deps struct {
	Config    *config.Config
	Switch    domain.SwitchController
	Store     *store.Store
	Publisher domain.Publisher
	Metrics   domain.MetricsRecorder
	Notifier  domain.ScreenPopNotifier
	AI        AIManager
	Analyzer  Analyzer
}
