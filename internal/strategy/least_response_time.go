package strategy

import (
	"errors"
	"sync"
	"time"

	"github.com/nikhil-thorat/relay/internal/target"
)

const ewmaAlpha = 0.2

type LeastResponseTime struct {
	mu   sync.Mutex
	ewma map[string]float64
}

func (l *LeastResponseTime) Observe(id string, d time.Duration) {
	ms := float64(d.Milliseconds())
	if ms < 1 {
		ms = float64(d.Nanoseconds()) / 1e6
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.ewma == nil {
		l.ewma = make(map[string]float64)
	}

	old, ok := l.ewma[id]
	if !ok {
		l.ewma[id] = ms
		return
	}

	l.ewma[id] = ewmaAlpha*ms + (1-ewmaAlpha)*old
}

func (l *LeastResponseTime) Select(targets []target.Target) (target.Target, error) {
	if len(targets) == 0 {
		return target.Target{}, errors.New("no target to select from")
	}

	if len(targets) == 1 {
		return targets[0], nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	best := targets[0]
	bestRT := l.ewmaValueLocked(best.ID)

	for _, t := range targets[1:] {
		rt := l.ewmaValueLocked(t.ID)
		if rt < bestRT {
			best = t
			bestRT = rt
		}
	}

	return best, nil
}

func (l *LeastResponseTime) ewmaValueLocked(id string) float64 {
	if l.ewma == nil {
		return 0
	}
	return l.ewma[id]
}
