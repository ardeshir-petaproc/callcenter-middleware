package ari

// RawChannel holds Asterisk channel payload from ARI.
type RawChannel struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	State    string      `json:"state"`
	Caller   RawCaller   `json:"caller"`
	Dialplan RawDialplan `json:"dialplan"`
}

// RawCaller holds caller ID details.
type RawCaller struct {
	Name   string `json:"name"`
	Number string `json:"number"`
}

// RawDialplan is the dialplan location of a channel.
type RawDialplan struct {
	Context  string `json:"context"`
	Exten    string `json:"exten"`
	Priority int    `json:"priority"`
}

// RawPlayback represents playback state in Asterisk ARI.
type RawPlayback struct {
	ID        string `json:"id"`
	MediaURI  string `json:"media_uri"`
	TargetURI string `json:"target_uri"`
	State     string `json:"state"`
}

// RawRecording is a live recording in Asterisk ARI.
type RawRecording struct {
	Name     string `json:"name"`
	State    string `json:"state"`
	Duration int    `json:"duration"`
	Cause    string `json:"cause"`
}

// RawEvent represents incoming Asterisk ARI WebSocket events.
type RawEvent struct {
	Type      string        `json:"type"`
	Timestamp string        `json:"timestamp"`
	Channel   *RawChannel   `json:"channel,omitempty"`
	Playback  *RawPlayback  `json:"playback,omitempty"`
	Recording *RawRecording `json:"recording,omitempty"`
	Digit     string        `json:"digit,omitempty"`
	Args      []string      `json:"args,omitempty"`
	Cause     int           `json:"cause,omitempty"`
	CauseTxt  string        `json:"cause_txt,omitempty"`
}
