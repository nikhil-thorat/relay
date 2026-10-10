package strategy

import (
	"errors"
	"sync"

	"github.com/nikhil-thorat/relay/internal/target"
)

type RoundRobin struct {
	mu      sync.Mutex
	current int
}

func (rr *RoundRobin) Select(targets []target.Target) (target.Target, error) {
	if len(targets) == 0 {
		return target.Target{}, errors.New("no target to select from")
	}

	if len(targets) == 1 {
		return targets[0], nil
	}

	rr.mu.Lock()
	defer rr.mu.Unlock()

	rr.current = rr.current % len(targets)
	selected := targets[rr.current]
	rr.current = (rr.current + 1) % len(targets)

	return selected, nil
}
