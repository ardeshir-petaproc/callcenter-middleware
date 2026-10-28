package server

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"callcenter-middleware/internal/domain"
)

func bad(msg string) error { return errStatus(http.StatusBadRequest, msg) }

func prepareQueue(q *domain.Queue) error {
	q.Name = strings.TrimSpace(q.Name)
	if q.Name == "" {
		return bad("name is required")
	}
	if q.Mode == "" {
		q.Mode = domain.QueueModeNative
	}
	if q.Mode != domain.QueueModeNative && q.Mode != domain.QueueModeAsterisk {
		return bad("mode must be native or asterisk")
	}
	if q.Mode == domain.QueueModeAsterisk && q.Extension == "" {
		return bad("asterisk queues need a dialplan extension")
	}
	switch q.Strategy {
	case "":
		q.Strategy = domain.StrategyLongestIdle
	case domain.StrategyRingAll, domain.StrategyLongestIdle, domain.StrategyRoundRobin,
		domain.StrategyFewestCalls, domain.StrategyRandom, domain.StrategyLinear:
	default:
		return bad("unknown strategy")
	}
	if q.RingTimeoutSec < 0 || q.MaxWaitSec < 0 || q.WrapupSec < 0 || q.SLASec < 0 {
		return bad("durations must be positive")
	}
	if q.SLASec == 0 {
		q.SLASec = 20
	}
	if q.RequiredSkills == nil {
		q.RequiredSkills = []string{}
	}
	return validAction(q.Overflow)
}

func validAction(a domain.Action) error {
	switch a.Type {
	case domain.ActionNone, domain.ActionVoicemail, domain.ActionCallback, domain.ActionHangup:
		return nil
	case domain.ActionQueue, domain.ActionAI, domain.ActionFlow, domain.ActionAgent,
		domain.ActionExternal, domain.ActionDialplan:
		if strings.TrimSpace(a.Target) == "" && a.Type != domain.ActionAI {
			return bad("action " + string(a.Type) + " needs a target")
		}
		return nil
	}
	return bad("unknown action type " + string(a.Type))
}

func prepareFlow(f *domain.IVRFlow) error {
	f.Name = strings.TrimSpace(f.Name)
	if f.Name == "" {
		return bad("name is required")
	}
	if f.TimeoutSec <= 0 {
		f.TimeoutSec = 5
	}
	if f.MaxRetries < 0 {
		f.MaxRetries = 0
	}
	if f.Options == nil {
		f.Options = map[string]domain.Action{}
	}
	for digit, a := range f.Options {
		if len(digit) != 1 || !strings.Contains("0123456789*#", digit) {
			return bad("invalid menu key " + digit)
		}
		if err := validAction(a); err != nil {
			return err
		}
	}
	for _, a := range []domain.Action{f.TimeoutAction, f.DirectAction, f.AfterHoursAction} {
		if err := validAction(a); err != nil {
			return err
		}
	}
	return nil
}

func prepareSchedule(s *domain.Schedule) error {
	s.Name = strings.TrimSpace(s.Name)
	if s.Name == "" {
		return bad("name is required")
	}
	if s.Timezone == "" {
		s.Timezone = "UTC"
	}
	if _, err := time.LoadLocation(s.Timezone); err != nil {
		return bad("unknown timezone")
	}
	if s.Weekly == nil {
		s.Weekly = []domain.TimeRange{}
	}
	if s.Holidays == nil {
		s.Holidays = []domain.Holiday{}
	}
	for _, r := range s.Weekly {
		if r.Day < 0 || r.Day > 6 || len(r.Start) != 5 || len(r.End) != 5 || r.Start >= r.End {
			return bad("invalid weekly range")
		}
	}
	for _, h := range s.Holidays {
		if _, err := time.Parse("2006-01-02", h.Date); err != nil {
			return bad("holiday dates must be YYYY-MM-DD")
		}
	}
	return nil
}

func prepareDisposition(d *domain.Disposition) error {
	d.Code, d.Label = strings.TrimSpace(d.Code), strings.TrimSpace(d.Label)
	if d.Code == "" {
		return bad("code is required")
	}
	if d.Label == "" {
		d.Label = d.Code
	}
	return nil
}

func preparePauseReason(p *domain.PauseReason) error {
	if p.Label = strings.TrimSpace(p.Label); p.Label == "" {
		return bad("label is required")
	}
	return nil
}

func prepareWebhook(h *domain.Webhook) error {
	u, err := url.Parse(h.URL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return bad("url must be http(s)")
	}
	if h.Events == nil {
		h.Events = []string{}
	}
	return nil
}

func prepareContact(c *domain.Contact) error {
	c.Name, c.Phone = strings.TrimSpace(c.Name), strings.TrimSpace(c.Phone)
	if c.Name == "" && c.Phone == "" {
		return bad("name or phone is required")
	}
	if c.Tags == nil {
		c.Tags = []string{}
	}
	return nil
}

func contactFilter(items []domain.Contact, q string) []domain.Contact {
	q = strings.ToLower(q)
	out := []domain.Contact{}
	for _, c := range items {
		hay := strings.ToLower(c.Name + " " + c.Phone + " " + c.Email + " " + c.Company + " " + strings.Join(c.Tags, " "))
		if strings.Contains(hay, q) {
			out = append(out, c)
		}
	}
	return out
}
