// Fake OpenAI Realtime server for interop testing with real Asterisk.
package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"sync/atomic"

	"github.com/coder/websocket"
)

func ulaw(sample int16) byte {
	const bias = 0x84
	s := int(sample)
	sign := 0
	if s < 0 {
		sign, s = 0x80, -s
	}
	if s > 32635 {
		s = 32635
	}
	s += bias
	exp := 7
	for mask := 0x4000; s&mask == 0 && exp > 0; mask >>= 1 {
		exp--
	}
	mant := (s >> (exp + 3)) & 0x0F
	return ^byte(sign | exp<<4 | mant)
}

var appends, sessions atomic.Int64

func main() {
	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]int64{"appends": appends.Load(), "sessions": sessions.Load()})
	})
	http.HandleFunc("/v1/realtime", func(w http.ResponseWriter, r *http.Request) {
		c, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		sessions.Add(1)
		ctx := r.Context()
		send := func(v any) { b, _ := json.Marshal(v); c.Write(ctx, websocket.MessageText, b) }
		n := 0
		for {
			_, msg, err := c.Read(ctx)
			if err != nil {
				log.Printf("session closed: %v (appends=%d)", err, n)
				return
			}
			var ev map[string]any
			json.Unmarshal(msg, &ev)
			switch ev["type"] {
			case "session.update":
				log.Printf("session.update: %s", string(msg)[:200])
			case "response.create":
				// 2 seconds of 440 Hz tone as the "greeting".
				audio := make([]byte, 16000)
				for i := range audio {
					audio[i] = ulaw(int16(8000 * math.Sin(2*math.Pi*440*float64(i)/8000)))
				}
				send(map[string]any{"type": "response.output_audio.delta", "item_id": "it1", "delta": base64.StdEncoding.EncodeToString(audio)})
				send(map[string]any{"type": "response.output_audio_transcript.done", "transcript": "Hello, how can I help?"})
			case "input_audio_buffer.append":
				n++
				appends.Add(1)
				if n == 30 {
					send(map[string]any{"type": "conversation.item.input_audio_transcription.completed", "transcript": "I need a human please"})
					send(map[string]any{"type": "response.function_call_arguments.done", "call_id": "fc1", "name": "transfer_to_queue",
						"arguments": `{"queue":"Support","reason":"asked for human","summary":"Caller wants to talk to a person."}`})
				}
			}
		}
	})
	log.Fatal(http.ListenAndServe(":18099", nil))
}
