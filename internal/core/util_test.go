package core_test

import (
	"strings"
	"testing"
	"time"

	"callcenter-middleware/internal/core"
	"callcenter-middleware/internal/domain"
)

// sscanf3 parses "play <ch> <uri> <id>" operations.
func sscanf3(op string, ch, uri, id *string) (int, error) {
	parts := strings.Fields(op)
	if len(parts) != 4 || parts[0] != "play" {
		return 0, nil
	}
	*ch, *uri, *id = parts[1], parts[2], parts[3]
	return 3, nil
}

func TestIsOpen(t *testing.T) {
	s := &domain.Schedule{
		Timezone: "UTC",
		Weekly:   []domain.TimeRange{{Day: 2, Start: "09:00", End: "17:00"}},
		Holidays: []domain.Holiday{{Date: "2026-12-29"}},
	}
	tue := time.Date(2026, 10, 27, 10, 0, 0, 0, time.UTC) // Tuesday
	if !core.IsOpen(s, tue) {
		t.Fatal("expected open on Tuesday 10:00")
	}
	if core.IsOpen(s, tue.Add(8*time.Hour)) {
		t.Fatal("expected closed at 18:00")
	}
	if core.IsOpen(s, tue.Add(24*time.Hour)) {
		t.Fatal("expected closed on Wednesday")
	}
	if core.IsOpen(s, time.Date(2026, 12, 29, 10, 0, 0, 0, time.UTC)) {
		t.Fatal("expected closed on holiday")
	}
	if !core.IsOpen(&domain.Schedule{}, tue) {
		t.Fatal("empty schedule should be open")
	}
}

func TestNormalizePhone(t *testing.T) {
	if got := core.NormalizePhone("+98 (912) 000-1234"); got != "989120001234" {
		t.Fatalf("got %s", got)
	}
	if got := core.NormalizePhoneKeepPlus(" +1 555 0100"); got != "+15550100" {
		t.Fatalf("got %s", got)
	}
}
