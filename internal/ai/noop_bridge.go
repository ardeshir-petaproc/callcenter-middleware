package ai

import (
	"context"

	"callcenter-middleware/internal/domain"
)

// NoopBridge provides a stub implementation of VoiceAgentBridge for Phase 1.
type NoopBridge struct{}

// NewNoopBridge constructs a stub voice agent bridge.
func NewNoopBridge() *NoopBridge {
	return &NoopBridge{}
}

// StartVoiceSession is a placeholder for Asterisk ExternalMedia audio socket bridge.
func (b *NoopBridge) StartVoiceSession(ctx context.Context, session *domain.CallSession) error {
	return nil
}

// StopVoiceSession terminates an active AI audio session.
func (b *NoopBridge) StopVoiceSession(ctx context.Context, channelID string) error {
	return nil
}
