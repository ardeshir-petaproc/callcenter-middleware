package ai

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
)

// RecordingFetcher downloads stored Asterisk recordings.
type RecordingFetcher interface {
	FetchRecording(ctx context.Context, name string) (*http.Response, error)
}

// Analyzer produces post-call summaries, sentiment and transcripts.
type Analyzer struct {
	Store   *store.Store
	OpenAI  *OpenAI
	Fetcher RecordingFetcher
	Pub     domain.Publisher
}

type analysis struct {
	Summary     string `json:"summary"`
	Sentiment   string `json:"sentiment"`
	Disposition string `json:"suggested_disposition"`
}

const analysisPrompt = `You analyze call center conversations. Reply with JSON:
{"summary": "<2-4 sentence summary for the CRM>", "sentiment": "positive|neutral|negative",
 "suggested_disposition": "<short snake_case outcome label>"}`

// Analyze summarizes a finished call. Calls without a transcript are
// transcribed from their recording first when one exists.
func (a *Analyzer) Analyze(ctx context.Context, callID string) {
	if a.OpenAI == nil || a.OpenAI.cfg.OpenAIAPIKey == "" {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	call, err := a.Store.GetCall(ctx, callID)
	if err != nil {
		return
	}
	lines, _ := a.Store.Transcript(ctx, callID)
	if len(lines) == 0 && call.Recording != "" && a.Fetcher != nil {
		// Give Asterisk a moment to finalize the file after hangup.
		time.Sleep(2 * time.Second)
		if text, err := a.transcribeRecording(ctx, call.Recording); err == nil && text != "" {
			t := domain.TranscriptEntry{CallID: callID, Speaker: "call", Text: text, At: time.Now().UTC()}
			_ = a.Store.AddTranscript(ctx, t)
			lines = append(lines, t)
		} else if err != nil {
			slog.Warn("Recording transcription failed", "call_id", callID, "err", err)
		}
	}
	if len(lines) == 0 {
		return
	}
	var b strings.Builder
	for _, l := range lines {
		fmt.Fprintf(&b, "%s: %s\n", l.Speaker, l.Text)
	}
	var res analysis
	if err := a.OpenAI.ChatJSON(ctx, analysisPrompt, b.String(), &res); err != nil {
		slog.Warn("Call analysis failed", "call_id", callID, "err", err)
		return
	}
	summary := res.Summary
	if strings.HasPrefix(call.Summary, "AI handoff:") {
		summary = call.Summary + "\n" + summary
	}
	if err := a.Store.UpdateCallAnalysis(ctx, callID, summary, res.Sentiment); err != nil {
		return
	}
	if call.Disposition == "" && res.Disposition != "" {
		_ = a.Store.UpdateCallWrapup(ctx, callID, "ai:"+res.Disposition, call.Notes, call.Tags)
	}
	if a.Pub != nil {
		if updated, err := a.Store.GetCall(ctx, callID); err == nil {
			a.Pub.Publish(domain.Event{Type: domain.EvCallEnded, Data: updated})
		}
	}
}

func (a *Analyzer) transcribeRecording(ctx context.Context, name string) (string, error) {
	resp, err := a.Fetcher.FetchRecording(ctx, name)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	return a.OpenAI.Transcribe(ctx, name+".wav", resp.Body)
}

// TranscribeVoicemail fills in the transcript of a stored voicemail.
func (a *Analyzer) TranscribeVoicemail(ctx context.Context, vm domain.Voicemail) {
	if a.OpenAI == nil || a.OpenAI.cfg.OpenAIAPIKey == "" || a.Fetcher == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	text, err := a.transcribeRecording(ctx, vm.Recording)
	if err != nil {
		slog.Warn("Voicemail transcription failed", "id", vm.ID, "err", err)
		return
	}
	vm.Transcript = text
	_ = a.Store.UpdateVoicemail(ctx, &vm)
}
