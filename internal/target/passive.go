package target

import "errors"

const maxFailures = 3

func (pool *Pool) RecordFailure(ID string) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	state, ok := pool.states[ID]
	if !ok {
		return errors.New("state not found")
	}

	state.ConsecutiveFailures++
	if state.ConsecutiveFailures >= maxFailures {
		state.Healthy = false
	}
	return nil
}

func (pool *Pool) RecordSuccess(ID string) error {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	state, ok := pool.states[ID]
	if !ok {
		return errors.New("state not found")
	}

	state.ConsecutiveFailures = 0
	return nil
}
