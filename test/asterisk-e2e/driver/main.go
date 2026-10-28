// Drives a real Asterisk + middleware end-to-end scenario.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/coder/websocket"
)

const ari = "http://127.0.0.1:18089/ari"
const api = "http://127.0.0.1:18080/api/v1"

var events = make(chan map[string]any, 100)

func ariDo(method, path string) (int, string) {
	req, _ := http.NewRequest(method, ari+path, nil)
	req.SetBasicAuth("tester", "tester")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

func apiDo(method, path, tok string, body any, out any) int {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, api+path, &buf)
	if tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fail("api %s: %v", path, err)
	}
	defer resp.Body.Close()
	if out != nil {
		json.NewDecoder(resp.Body).Decode(out)
	}
	return resp.StatusCode
}

func fail(f string, a ...any) { fmt.Printf("FAIL: "+f+"\n", a...); os.Exit(1) }
func pass(f string, a ...any) { fmt.Printf("PASS: "+f+"\n", a...) }

func login(u, p string) string {
	var r struct{ Token string }
	apiDo("POST", "/auth/login", "", map[string]string{"username": u, "password": p}, &r)
	if r.Token == "" {
		fail("login %s", u)
	}
	return r.Token
}

func liveCalls(tok string) []map[string]any {
	var s struct{ Calls []map[string]any }
	apiDo("GET", "/live", tok, nil, &s)
	return s.Calls
}

func waitCall(tok, state string, d time.Duration) map[string]any {
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		for _, c := range liveCalls(tok) {
			if c["state"] == state {
				return c
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	fail("no call in state %s; live=%v", state, liveCalls(tok))
	return nil
}

func waitStasis(id string) {
	t := time.After(10 * time.Second)
	for {
		select {
		case ev := <-events:
			if ev["type"] == "StasisStart" {
				if ch, _ := ev["channel"].(map[string]any); ch["id"] == id {
					return
				}
			}
		case <-t:
			fail("tester channel %s never entered Stasis", id)
		}
	}
}

func setAgent(tok string, id int, endpoint string) {
	var u map[string]any
	apiDo("GET", fmt.Sprintf("/users/%d", id), tok, nil, &u)
	u["endpoint"] = endpoint
	if code := apiDo("PUT", fmt.Sprintf("/users/%d", id), tok, u, nil); code != 200 {
		fail("update user %d: %d", id, code)
	}
}

func main() {
	ctx := context.Background()
	conn, _, err := websocket.Dial(ctx, "ws://127.0.0.1:18089/ari/events?app=tester",
		&websocket.DialOptions{HTTPHeader: http.Header{"Authorization": []string{"Basic dGVzdGVyOnRlc3Rlcg=="}}})
	if err != nil {
		fail("tester ws: %v", err)
	}
	go func() {
		for {
			_, msg, err := conn.Read(ctx)
			if err != nil {
				return
			}
			var ev map[string]any
			json.Unmarshal(msg, &ev)
			events <- ev
		}
	}()
	admin := login("admin", "admin123")
	setAgent(admin, 2, "Local/answer@test-agents") // alice: Sales
	setAgent(admin, 3, "Local/answer@test-agents") // bob: Sales/Support
	if os.Args[1] == "survey" {
		setAgent(admin, 3, "Local/answer6@test-agents")
	}
	setAgent(admin, 6, "Local/answer@test-agents") // sara: supervisor
	alice, bob := login("alice", "agent123"), login("bob", "agent123")
	apiDo("POST", "/me/status", alice, map[string]string{"status": "available"}, nil)
	switch os.Args[1] {
	case "queue":
		scenarioQueue(admin, alice)
	case "voicemail":
		apiDo("POST", "/me/status", alice, map[string]string{"status": "offline"}, nil)
		scenarioVoicemail(admin)
	case "outbound":
		apiDo("POST", "/me/status", alice, map[string]string{"status": "offline"}, nil)
		scenarioOutbound(admin, alice)
	case "survey":
		apiDo("POST", "/me/status", alice, map[string]string{"status": "offline"}, nil)
		scenarioSurvey(admin, bob)
	case "campaign":
		scenarioCampaign(admin)
	case "consult":
		apiDo("POST", "/me/status", bob, map[string]string{"status": "offline"}, nil)
		scenarioConsult(admin, alice, bob)
	case "ai":
		apiDo("POST", "/me/status", bob, map[string]string{"status": "available"}, nil)
		apiDo("POST", "/me/status", alice, map[string]string{"status": "offline"}, nil)
		scenarioAI(admin)
	}
}

func dial(id, exten string) {
	code, body := ariDo("POST", "/channels?endpoint=Local/"+exten+"@from-internal&app=tester&channelId="+id+"&callerId=%2B15550100")
	if code != 200 {
		fail("originate test caller: %d %s", code, body)
	}
	waitStasis(id)
}
