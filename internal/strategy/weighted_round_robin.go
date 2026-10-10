package strategy

import (
	"errors"
	"sync"

	"github.com/nikhil-thorat/relay/internal/target"
)

type WeightedRoundRobin struct {
	mu      sync.Mutex
	current map[string]int
}

func (w *WeightedRoundRobin) Select(targets []target.Target) (target.Target, error) {
	if len(targets) == 0 {
		return target.Target{}, errors.New("no target to select from")
	}

	if len(targets) == 1 {
		return targets[0], nil
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if w.current == nil {
		w.current = make(map[string]int)
	}

	total := 0
	for _, t := range targets {
		weight := t.Weight
		if weight <= 0 {
			weight = 1
		}
		total += weight
	}

	var best target.Target
	bestWeight := -1
	first := true

	for _, t := range targets {
		weight := t.Weight
		if weight <= 0 {
			weight = 1
		}
		w.current[t.ID] += weight
		if first || w.current[t.ID] > bestWeight {
			best = t
			bestWeight = w.current[t.ID]
			first = false
		}
	}

	w.current[best.ID] -= total

	return best, nil
}
