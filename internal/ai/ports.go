package ai

import (
	"errors"
	"sync"
)

// ErrNoPorts is returned when every media port is in use.
var ErrNoPorts = errors.New("no free media ports")

// portPool hands out UDP ports from a fixed range for RTP sessions.
type portPool struct {
	mu    sync.Mutex
	min   int
	max   int
	next  int
	inUse map[int]bool
}

func newPortPool(min, max int) *portPool {
	if max < min {
		max = min
	}
	return &portPool{min: min, max: max, next: min, inUse: map[int]bool{}}
}

// acquire tries each port at most once, using try to bind it.
func (p *portPool) acquire(try func(port int) error) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	size := p.max - p.min + 1
	for i := 0; i < size; i++ {
		port := p.next
		p.next++
		if p.next > p.max {
			p.next = p.min
		}
		if p.inUse[port] {
			continue
		}
		if err := try(port); err != nil {
			continue
		}
		p.inUse[port] = true
		return port, nil
	}
	return 0, ErrNoPorts
}

func (p *portPool) release(port int) {
	p.mu.Lock()
	delete(p.inUse, port)
	p.mu.Unlock()
}
