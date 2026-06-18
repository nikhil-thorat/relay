package target

import "errors"

type Pool struct {
	Targets map[string]*Target
	States  map[string]*TargetState
}

func NewPool() *Pool {
	return &Pool{
		Targets: make(map[string]*Target),
		States:  make(map[string]*TargetState),
	}
}

func (pool *Pool) Add(target *Target) error {
	_, ok := pool.Targets[target.ID]
	if ok {
		return errors.New("target already exists")
	}

	pool.Targets[target.ID] = target
	pool.States[target.ID] = &TargetState{
		Healthy: false,
	}

	return nil
}

func (pool *Pool) Get(ID string) (*Target, error) {
	target, ok := pool.Targets[ID]
	if !ok {
		return nil, errors.New("target not found")
	}

	return target, nil
}

func (pool *Pool) List() []*Target {
	var targets []*Target

	for _, target := range pool.Targets {
		targets = append(targets, target)
	}

	return targets
}
