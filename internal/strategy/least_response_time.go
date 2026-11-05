package strategy

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/nikhil-thorat/relay/internal/target"
)

const ewmaAlpha = 0.2

type LeastResponseTime struct {
	mu      sync.Mutex
	counter int
	ewma    map[string]float64
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

	var unknown []target.Target
	for _, t := range targets {
		if math.IsInf(l.ewmaValueLocked(t.ID), 1) {
			unknown = append(unknown, t)
		}
	}

	if len(unknown) > 0 {
		best := unknown[l.counter%len(unknown)]
		l.counter++
		return best, nil
	}

	var best target.Target
	bestRT := math.Inf(1)
	count := 0

	for _, t := range targets {
		rt := l.ewmaValueLocked(t.ID)
		if rt < bestRT {
			best = t
			bestRT = rt
			count = 1
			continue
		}
		if rt == bestRT {
			count++
		}
	}

	if count == 1 {
		return best, nil
	}

	idx := l.counter % count
	l.counter++

	for _, t := range targets {
		if l.ewmaValueLocked(t.ID) == bestRT {
			if idx == 0 {
				return t, nil
			}
			idx--
		}
	}

	return best, nil
}

func (l *LeastResponseTime) Prune(targets []target.Target) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.ewma == nil {
		return
	}

	active := make(map[string]struct{}, len(targets))
	for _, t := range targets {
		active[t.ID] = struct{}{}
	}

	for id := range l.ewma {
		if _, ok := active[id]; !ok {
			delete(l.ewma, id)
		}
	}
}

func (l *LeastResponseTime) ewmaValueLocked(id string) float64 {
	if l.ewma == nil {
		return math.Inf(1)
	}

	value, ok := l.ewma[id]
	if !ok {
		return math.Inf(1)
	}

	return value
}
