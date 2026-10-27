package sim

import (
	"context"
	"math/rand/v2"
	"time"
)

// RunGenerator creates inbound calls at the configured average rate.
func (s *Sim) RunGenerator(ctx context.Context) {
	if s.opts.CallsPerMinute <= 0 {
		return
	}
	for {
		mean := 60 / s.opts.CallsPerMinute
		wait := time.Duration(rand.ExpFloat64() * mean / s.opts.Speed * float64(time.Second))
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		s.mu.Lock()
		if len(s.clients) > 0 {
			s.Inbound(SimCall{Digits: s.randomDigits()})
		}
		s.mu.Unlock()
	}
}

// randomDigits picks a menu choice; some callers never press anything and
// some press an invalid key first.
func (s *Sim) randomDigits() string {
	if s.opts.Digits == "" || s.rng.Float64() < 0.12 {
		return ""
	}
	d := string(s.opts.Digits[s.rng.IntN(len(s.opts.Digits))])
	if s.rng.Float64() < 0.08 {
		return "7" + d
	}
	return d
}
