package target

import (
	"errors"
	"sync"
)

type Pool struct {
	mu sync.RWMutex

	Targets map[string]*Target
	States  map[string]*TargetState

	Order []string
}

func NewPool() *Pool {
	return &Pool{
		Targets: make(map[string]*Target),
		States:  make(map[string]*TargetState),
		Order:   make([]string, 0),
	}
}

func (pool *Pool) Add(target *Target) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	_, ok := pool.Targets[target.ID]
	if ok {
		return errors.New("target already exists")
	}

	pool.Targets[target.ID] = target
	pool.States[target.ID] = &TargetState{
		Healthy: false,
	}
	pool.Order = append(pool.Order, target.ID)

	return nil
}

func (pool *Pool) Get(ID string) (*Target, error) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	target, ok := pool.Targets[ID]
	if !ok {
		return nil, errors.New("target not found")
	}

	return target, nil
}

func (pool *Pool) GetState(ID string) (TargetState, error) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	state, ok := pool.States[ID]
	if !ok {
		return TargetState{}, errors.New("state not found")
	}

	return *state, nil
}

func (pool *Pool) List() []*Target {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	targets := make([]*Target, 0, len(pool.Order))

	for _, ID := range pool.Order {
		targets = append(targets, pool.Targets[ID])
	}

	return targets
}

func (pool *Pool) SetHealthy(ID string, healthy bool) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	state, ok := pool.States[ID]
	if !ok {
		return errors.New("state not found")
	}

	state.Healthy = healthy
	return nil
}

func (pool *Pool) Healthy() []*Target {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	targets := make([]*Target, 0, len(pool.Order))

	for _, ID := range pool.Order {
		if pool.States[ID].Healthy {
			targets = append(targets, pool.Targets[ID])
		}
	}

	return targets
}
