package strategy

import (
	"errors"
	"sync"
	"sync/atomic"

	"github.com/nikhil-thorat/relay/internal/target"
)

type LeastConnections struct {
	mu       sync.Mutex
	next     int
	inflight map[string]*atomic.Int64
}

func (l *LeastConnections) counter(id string) *atomic.Int64 {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.counterLocked(id)
}

func (l *LeastConnections) counterLocked(id string) *atomic.Int64 {
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
	l.mu.Lock()
	defer l.mu.Unlock()

	l.counterLocked(id).Add(1)
}

func (l *LeastConnections) Done(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.counterLocked(id).Add(-1)
}

func (l *LeastConnections) Select(targets []target.Target) (target.Target, error) {
	if len(targets) == 0 {
		return target.Target{}, errors.New("no target to select from")
	}

	if len(targets) == 1 {
		return targets[0], nil
	}

	l.mu.Lock()
	start := l.next % len(targets)
	l.next = (start + 1) % len(targets)
	l.mu.Unlock()

	best := targets[start]
	bestCount := l.counter(best.ID).Load()

	for i := 1; i < len(targets); i++ {
		t := targets[(start+i)%len(targets)]
		count := l.counter(t.ID).Load()
		if count < bestCount {
			best = t
			bestCount = count
		}
	}

	return best, nil
}

func (l *LeastConnections) Prune(targets []target.Target) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.inflight == nil {
		return
	}

	active := make(map[string]struct{}, len(targets))
	for _, t := range targets {
		active[t.ID] = struct{}{}
	}

	for id, counter := range l.inflight {
		if _, ok := active[id]; !ok && counter.Load() == 0 {
			delete(l.inflight, id)
		}
	}
}
