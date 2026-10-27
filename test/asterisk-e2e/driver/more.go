package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func waitHistory(tok, query string, d time.Duration) map[string]any {
	end := time.Now().Add(d)
	for time.Now().Before(end) {
		var res struct{ Items []map[string]any }
		apiDo("GET", "/calls?"+query, tok, nil, &res)
		if len(res.Items) > 0 {
			return res.Items[0]
		}
		time.Sleep(300 * time.Millisecond)
	}
	fail("no call history for %s", query)
	return nil
}

func scenarioVoicemail(admin string) {
	dial("tc-3", "100")
	waitCall(admin, "ivr", 5*time.Second)
	time.Sleep(300 * time.Millisecond)
	ariDo("POST", "/channels/tc-3/dtmf?dtmf=0")
	waitCall(admin, "voicemail", 5*time.Second)
	ariDo("POST", "/channels/tc-3/play?media=tone:ring")
	time.Sleep(4 * time.Second)
	ariDo("DELETE", "/channels/tc-3")
	var vms []map[string]any
	end := time.Now().Add(5 * time.Second)
	for time.Now().Before(end) && len(vms) == 0 {
		apiDo("GET", "/voicemails", admin, nil, &vms)
		time.Sleep(300 * time.Millisecond)
	}
	if len(vms) == 0 {
		fail("no voicemail stored")
	}
	id := fmt.Sprint(vms[0]["id"])
	resp, err := http.Get(api + "/voicemails/" + id + "/audio?token=" + admin)
	if err != nil || resp.StatusCode != 200 {
		fail("voicemail audio: %v %v", err, resp)
	}
	n, _ := io.Copy(io.Discard, resp.Body)
	c := waitHistory(admin, "outcome=voicemail", 3*time.Second)
	pass("voicemail recorded (%vs, %d bytes audio), CDR outcome=%v", vms[0]["duration_sec"], n, c["outcome"])
}

func scenarioOutbound(admin, alice string) {
	var res struct {
		CallID string `json:"call_id"`
	}
	if code := apiDo("POST", "/me/dial", alice, map[string]string{"number": "+1 555 0101"}, &res); code != 200 {
		fail("dial: %d", code)
	}
	waitCall(admin, "connected", 10*time.Second)
	c := waitHistory(admin, "direction=outbound&outcome=answered", 15*time.Second)
	pass("click-to-call: agent then customer bridged; CDR %v talk=%vs contact=%v", c["outcome"], c["talk_sec"], c["caller_name"])
}

func scenarioSurvey(admin, bob string) {
	apiDo("POST", "/me/status", bob, map[string]string{"status": "available"}, nil)
	dial("tc-4", "100")
	waitCall(admin, "ivr", 5*time.Second)
	time.Sleep(300 * time.Millisecond)
	ariDo("POST", "/channels/tc-4/dtmf?dtmf=2")
	waitCall(admin, "connected", 10*time.Second)
	waitCall(admin, "survey", 12*time.Second) // agent dialplan hangs up after 6s
	time.Sleep(500 * time.Millisecond)
	ariDo("POST", "/channels/tc-4/dtmf?dtmf=5")
	c := waitHistory(admin, "q=%2B15550100&outcome=answered&queue_id=14", 10*time.Second)
	for i := 0; i < 20 && fmt.Sprint(c["csat"]) != "5"; i++ {
		time.Sleep(300 * time.Millisecond)
		c = waitHistory(admin, "q=%2B15550100&outcome=answered&queue_id=14", time.Second)
	}
	if fmt.Sprint(c["csat"]) != "5" {
		fail("expected csat 5, got %v", c)
	}
	pass("post-call survey on real Asterisk: csat=%v talk=%vs", c["csat"], c["talk_sec"])
}

func scenarioCampaign(admin string) {
	var cps []map[string]any
	apiDo("GET", "/campaigns", admin, nil, &cps)
	id := fmt.Sprint(cps[0]["id"])
	apiDo("POST", "/campaigns/"+id+"/start", admin, nil, nil)
	end := time.Now().Add(60 * time.Second)
	for time.Now().Before(end) {
		var st struct{ Leads map[string]float64 }
		apiDo("GET", "/campaigns/"+id+"/stats", admin, nil, &st)
		if st.Leads["answered"] == st.Leads["total"] {
			pass("campaign dialed all %v leads on real Asterisk: %v", st.Leads["total"], st.Leads)
			return
		}
		time.Sleep(time.Second)
	}
	fail("campaign did not finish")
}
