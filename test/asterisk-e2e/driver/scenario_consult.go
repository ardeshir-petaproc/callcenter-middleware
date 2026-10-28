package main

import (
	"fmt"
	"time"
)

func waitConsult(tok, id, want string, d time.Duration) map[string]any {
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		for _, c := range liveCalls(tok) {
			if c["id"] == id && fmt.Sprint(c["consult"]) == want {
				return c
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	fail("call %s never reached consult=%q: %v", id, want, liveCalls(tok))
	return nil
}

// scenarioConsult: alice takes a queue call, consults an outside number that
// hangs up (caller returns to alice), then consults bob, conferences and
// hands the caller over to bob.
func scenarioConsult(admin, alice, bob string) {
	dial("tc-6", "100")
	waitCall(admin, "ivr", 5*time.Second)
	time.Sleep(500 * time.Millisecond)
	ariDo("POST", "/channels/tc-6/dtmf?dtmf=1")
	c := waitCall(admin, "connected", 10*time.Second)
	id := c["id"].(string)
	pass("alice connected to the caller")
	if code := apiDo("POST", "/calls/"+id+"/consult", alice, map[string]string{"type": "external", "target": "+15557777"}, nil); code != 200 {
		fail("consult external: %d", code)
	}
	waitConsult(admin, id, "talking", 10*time.Second)
	waitCall(admin, "hold", 2*time.Second)
	pass("caller on hold while alice consults an outside number")
	back := waitConsult(admin, id, "<nil>", 15*time.Second)
	if back["state"] != "connected" || back["agent_id"].(float64) != 2 {
		fail("caller should be back with alice: %v", back)
	}
	pass("outside party hung up: caller returned to alice")

	apiDo("POST", "/me/status", bob, map[string]string{"status": "available"}, nil)
	if code := apiDo("POST", "/calls/"+id+"/consult", alice, map[string]string{"type": "agent", "target": "3"}, nil); code != 200 {
		fail("consult bob: %d", code)
	}
	waitConsult(admin, id, "talking", 10*time.Second)
	if code := apiDo("POST", "/calls/"+id+"/consult/conference", alice, nil, nil); code != 200 {
		fail("conference: %d", code)
	}
	waitConsult(admin, id, "conference", 3*time.Second)
	pass("three-way conference: caller, alice and bob")
	if code := apiDo("POST", "/calls/"+id+"/consult/complete", alice, nil, nil); code != 200 {
		fail("complete: %d", code)
	}
	time.Sleep(time.Second)
	for _, lc := range liveCalls(admin) {
		if lc["id"] == id && (lc["agent_id"].(float64) != 3 || lc["state"] != "connected") {
			fail("bob should have the caller: %v", lc)
		}
	}
	pass("transfer completed: bob has the caller, alice released")
	ariDo("DELETE", "/channels/tc-6")
	var detail struct{ Events []map[string]any }
	time.Sleep(2 * time.Second)
	apiDo("GET", "/calls/"+id, admin, nil, &detail)
	var types []string
	for _, e := range detail.Events {
		types = append(types, e["type"].(string))
	}
	pass("timeline: %v", types)
}
