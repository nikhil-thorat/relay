package strategy

import (
	"errors"

	"github.com/nikhil-thorat/relay/internal/target"
)

type RoundRobin struct {
	current int
}

func (rr *RoundRobin) Select(targets []*target.Target) (*target.Target, error) {
	if len(targets) == 0 {
		return nil, errors.New("no target to select from")
	}

	if len(targets) == 1 {
		return targets[0], nil
	}

	selected := targets[rr.current]
	rr.current = (rr.current + 1) % len(targets)

	return selected, nil
}
