package main

import (
	"io"
	"net/http"
	"strings"
	"time"
)

func scenarioQueue(admin, alice string) {
	dial("tc-1", "100")
	c := waitCall(admin, "ivr", 5*time.Second)
	pass("inbound call entered IVR (caller=%v, contact=%v)", c["caller_number"], c["caller_name"])
	time.Sleep(500 * time.Millisecond)
	if code, b := ariDo("POST", "/channels/tc-1/dtmf?dtmf=1"); code != 204 {
		fail("send dtmf: %d %s", code, b)
	}
	c = waitCall(admin, "connected", 10*time.Second)
	pass("DTMF 1 -> Sales -> alice answered, bridged (agent_id=%v recording=%v)", c["agent_id"], c["recording"])
	id := c["id"].(string)
	if code := apiDo("POST", "/calls/"+id+"/hold", alice, map[string]bool{"on": true}, nil); code != 200 {
		fail("hold %d", code)
	}
	waitCall(admin, "hold", 3*time.Second)
	apiDo("POST", "/calls/"+id+"/hold", alice, map[string]bool{"on": false}, nil)
	waitCall(admin, "connected", 3*time.Second)
	pass("hold/unhold via API")
	if code := apiDo("POST", "/calls/"+id+"/monitor", login("sara", "agent123"), map[string]string{"mode": "whisper"}, nil); code != 200 {
		fail("monitor %d", code)
	}
	time.Sleep(2 * time.Second)
	for _, lc := range liveCalls(admin) {
		if lc["id"] == id && lc["monitors"].(float64) != 1 {
			fail("expected 1 monitor, got %v", lc["monitors"])
		}
	}
	pass("supervisor whisper monitor via snoop")
	time.Sleep(2 * time.Second)
	ariDo("DELETE", "/channels/tc-1")
	time.Sleep(2 * time.Second)
	var detail struct {
		Call   map[string]any
		Events []map[string]any
	}
	apiDo("GET", "/calls/"+id, admin, nil, &detail)
	if detail.Call["outcome"] != "answered" {
		fail("expected answered outcome: %v", detail.Call)
	}
	var types []string
	for _, e := range detail.Events {
		types = append(types, e["type"].(string))
	}
	pass("CDR outcome=answered talk=%vs hold=%vs timeline=%s", detail.Call["talk_sec"], detail.Call["hold_sec"], strings.Join(types, ","))
	req, _ := http.NewRequest("GET", api+"/calls/"+id+"/recording?token="+admin, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 {
		fail("recording download: %v %v", err, resp)
	}
	n, _ := io.Copy(io.Discard, resp.Body)
	pass("recording %v downloaded via API proxy (%d bytes, %s)", detail.Call["recording"], n, resp.Header.Get("Content-Type"))
}

func scenarioAI(admin string) {
	dial("tc-2", "700")
	if code, b := ariDo("POST", "/channels/tc-2/play?media=tone:ring"); code != 201 && code != 200 {
		fail("caller tone: %d %s", code, b)
	}
	c := waitCall(admin, "ai", 5*time.Second)
	pass("DNIS 700 -> AI agent session started")
	c = waitCall(admin, "connected", 15*time.Second)
	pass("AI transferred to Support and bob answered; summary=%q", c["summary"])
	var detail struct{ Transcript []map[string]any }
	apiDo("GET", "/calls/"+c["id"].(string), admin, nil, &detail)
	if len(detail.Transcript) < 2 {
		fail("expected transcript, got %v", detail.Transcript)
	}
	pass("transcript: %v", detail.Transcript)
	ariDo("DELETE", "/channels/tc-2")
}
