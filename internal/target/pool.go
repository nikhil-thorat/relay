package target

import (
	"errors"
	"sync"
	"time"
)

type Pool struct {
	mu sync.RWMutex

	targets map[string]Target
	states  map[string]*TargetState

	order []string
}

func NewPool() *Pool {
	return &Pool{
		targets: make(map[string]Target),
		states:  make(map[string]*TargetState),
		order:   make([]string, 0),
	}
}

func (pool *Pool) Add(target *Target) error {
	if target == nil {
		return errors.New("cannot add nil target")
	}

	pool.mu.Lock()
	defer pool.mu.Unlock()

	_, ok := pool.targets[target.ID]
	if ok {
		return errors.New("target already exists")
	}

	pool.targets[target.ID] = *target
	pool.states[target.ID] = &TargetState{
		Healthy: false,
	}
	pool.order = append(pool.order, target.ID)

	return nil
}

func (pool *Pool) Remove(ID string) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	_, ok := pool.targets[ID]
	if !ok {
		return errors.New("target not found")
	}

	delete(pool.targets, ID)
	delete(pool.states, ID)

	for i, id := range pool.order {
		if id == ID {
			pool.order = append(pool.order[:i], pool.order[i+1:]...)
			break
		}
	}

	return nil
}

func (pool *Pool) Update(t *Target) error {
	if t == nil {
		return errors.New("cannot update nil target")
	}

	pool.mu.Lock()
	defer pool.mu.Unlock()

	_, ok := pool.targets[t.ID]
	if !ok {
		return errors.New("target not found")
	}

	pool.targets[t.ID] = *t
	return nil
}

func (pool *Pool) Get(ID string) (Target, error) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	target, ok := pool.targets[ID]
	if !ok {
		return Target{}, errors.New("target not found")
	}

	return target, nil
}

func (pool *Pool) GetState(ID string) (TargetState, error) {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	state, ok := pool.states[ID]
	if !ok {
		return TargetState{}, errors.New("state not found")
	}

	return *state, nil
}

func (pool *Pool) List() []Target {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	targets := make([]Target, 0, len(pool.order))

	for _, ID := range pool.order {
		targets = append(targets, pool.targets[ID])
	}

	return targets
}

func (pool *Pool) SetHealthy(ID string, healthy bool) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	state, ok := pool.states[ID]
	if !ok {
		return errors.New("state not found")
	}

	state.Healthy = healthy
	state.LastChecked = time.Now()
	if healthy {
		state.ConsecutiveFailures = 0
	}
	return nil
}

func (pool *Pool) Healthy() []Target {
	pool.mu.RLock()
	defer pool.mu.RUnlock()

	targets := make([]Target, 0, len(pool.order))

	for _, ID := range pool.order {
		if pool.states[ID].Healthy {
			targets = append(targets, pool.targets[ID])
		}
	}

	return targets
}
