package core

import (
	"time"

	"callcenter-middleware/internal/domain"
)

// IsOpen reports whether a schedule is open at t. An empty weekly list means
// always open (only holidays close it).
func IsOpen(s *domain.Schedule, t time.Time) bool {
	if s == nil {
		return true
	}
	loc := time.UTC
	if s.Timezone != "" {
		if l, err := time.LoadLocation(s.Timezone); err == nil {
			loc = l
		}
	}
	local := t.In(loc)
	date := local.Format("2006-01-02")
	for _, h := range s.Holidays {
		if h.Date == date {
			return false
		}
	}
	if len(s.Weekly) == 0 {
		return true
	}
	minute := local.Hour()*60 + local.Minute()
	for _, r := range s.Weekly {
		if r.Day != int(local.Weekday()) {
			continue
		}
		start, ok1 := parseHHMM(r.Start)
		end, ok2 := parseHHMM(r.End)
		if !ok1 || !ok2 {
			continue
		}
		if minute >= start && minute < end {
			return true
		}
	}
	return false
}

func parseHHMM(s string) (int, bool) {
	if s == "24:00" {
		return 24 * 60, true
	}
	t, err := time.Parse("15:04", s)
	if err != nil {
		return 0, false
	}
	return t.Hour()*60 + t.Minute(), true
}
