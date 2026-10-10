package strategy

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/nikhil-thorat/relay/internal/target"
)

type LeastConnections struct {
	mu       sync.Mutex
	inflight map[string]*atomic.Int64
}

func (l *LeastConnections) counter(id string) *atomic.Int64 {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.inflight == nil {
		l.inflight = make(map[string]*atomic.Int64)
	}

	c, ok := l.inflight[id]
	if !ok {
		c = &atomic.Int64{}
		l.inflight[id] = c
	}

	return c
}

func (l *LeastConnections) Inc(id string) {
	l.counter(id).Add(1)
}

func (l *LeastConnections) Done(id string) {
	l.counter(id).Add(-1)
}

func (l *LeastConnections) Select(targets []target.Target) (target.Target, error) {
	if len(targets) == 0 {
		return target.Target{}, errors.New("no target to select from")
	}

	if len(targets) == 1 {
		return targets[0], nil
	}

	best := targets[0]
	bestCount := l.counter(best.ID).Load()

	for _, t := range targets[1:] {
		count := l.counter(t.ID).Load()
		if count < bestCount {
			best = t
			bestCount = count
		}
	}

	return best, nil
}
