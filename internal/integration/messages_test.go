package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"callcenter-middleware/internal/domain"
)

func TestInboundMessages_ThreadIntoTickets(t *testing.T) {
	s := newStack(t, 40)
	admin := s.login("admin", "admin123")
	var issued struct{ Key, Endpoint string }
	if code := s.call("POST", "/integrations/inbound-key", admin, nil, &issued); code != 200 || issued.Key == "" {
		t.Fatalf("key: %d %+v", code, issued)
	}
	post := func(org, key string, m map[string]string) (int, map[string]any) {
		raw, _ := json.Marshal(m)
		req, _ := http.NewRequest("POST", s.api.URL+"/api/v1/inbound/messages?org="+org, bytes.NewReader(raw))
		req.Header.Set("X-Inbound-Key", key)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		var out map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&out)
		return resp.StatusCode, out
	}
	msg := map[string]string{"channel": "joow", "conversation_id": "conv-9", "from_name": "مریم", "from_id": "j-maryam",
		"text": "سفارشم هنوز نرسیده\nشماره سفارش ۱۲۳"}
	if code, _ := post("default", "wrong", msg); code != 401 {
		t.Fatalf("wrong key: %d", code)
	}
	if code, _ := post("parsian", issued.Key, msg); code != 401 {
		t.Fatalf("key of another organization: %d", code)
	}
	code, first := post("default", issued.Key, msg)
	if code != 200 || first["created"] != true {
		t.Fatalf("first message: %d %v", code, first)
	}
	msg["text"] = "Any news?"
	if code, again := post("default", issued.Key, msg); code != 200 || again["created"] != false || again["ticket_id"] != first["ticket_id"] {
		t.Fatalf("follow-up should thread: %d %v", code, again)
	}
	var detail struct {
		Ticket   domain.Ticket          `json:"ticket"`
		Comments []domain.TicketComment `json:"comments"`
	}
	id := int64(first["ticket_id"].(float64))
	s.call("POST", "/tickets/"+itoa(id)+"/comments", admin, map[string]any{"body": "It ships today."}, nil)
	s.call("GET", "/tickets/"+itoa(id), admin, nil, &detail)
	if detail.Ticket.Source != "message" || detail.Ticket.Subject != "سفارشم هنوز نرسیده" || len(detail.Comments) < 2 {
		t.Fatalf("ticket: %+v %+v", detail.Ticket, detail.Comments)
	}
	for _, c := range detail.Comments {
		if c.Internal {
			t.Fatalf("reply delivery should not fail with the local provider: %+v", c)
		}
	}
}
