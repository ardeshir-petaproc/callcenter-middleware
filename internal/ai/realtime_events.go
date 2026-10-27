package ai

import (
	"context"
	"encoding/base64"
	"errors"
	"log/slog"
)

func (s *Session) handle(ctx context.Context, ev rtEvent) error {
	switch ev.Type {
	case "response.output_audio.delta", "response.audio.delta":
		audio, err := base64.StdEncoding.DecodeString(ev.Delta)
		if err != nil {
			return nil
		}
		s.mu.Lock()
		if ev.ItemID != s.itemID {
			s.itemID = ev.ItemID
			s.itemStart = s.rtp.PlayedMS() + s.rtp.Pending().Milliseconds()
		}
		s.mu.Unlock()
		s.rtp.Enqueue(audio)
	case "input_audio_buffer.speech_started":
		s.bargeIn(ctx)
	case "conversation.item.input_audio_transcription.completed":
		if ev.Transcript != "" {
			s.ctl.AITranscript(s.spec.CallID, "caller", ev.Transcript)
		}
	case "response.output_audio_transcript.done", "response.audio_transcript.done":
		if ev.Transcript != "" {
			s.ctl.AITranscript(s.spec.CallID, "ai", ev.Transcript)
		}
	case "response.function_call_arguments.done":
		go s.handleTool(ctx, ev.CallID, ev.Name, ev.Arguments)
	case "error":
		if ev.Error != nil {
			slog.Warn("Realtime API error", "call_id", s.spec.CallID, "code", ev.Error.Code, "msg", ev.Error.Message)
			if ev.Error.Type == "invalid_request_error" && ev.Error.Code == "session_expired" {
				return errors.New("realtime session expired")
			}
		}
	}
	return nil
}

// bargeIn stops assistant playout when the caller starts speaking and tells
// the model how much of its answer was actually heard.
func (s *Session) bargeIn(ctx context.Context) {
	if s.rtp.Pending() == 0 || s.ending.Load() {
		return
	}
	s.mu.Lock()
	item, played := s.itemID, s.rtp.PlayedMS()-s.itemStart
	s.mu.Unlock()
	s.rtp.Flush()
	if item != "" && played >= 0 {
		_ = s.send(ctx, map[string]any{"type": "conversation.item.truncate", "item_id": item,
			"content_index": 0, "audio_end_ms": played})
	}
}
