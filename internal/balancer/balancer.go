package balancer

import (
	"sync"
	"time"

	"github.com/nikhil-thorat/relay/internal/strategy"
	"github.com/nikhil-thorat/relay/internal/target"
)

type Balancer struct {
	mu       sync.RWMutex
	pool     *target.Pool
	strategy strategy.Strategy
}

type Selection struct {
	Target   target.Target
	strategy strategy.Strategy
}

func New(pool *target.Pool, strategy strategy.Strategy) *Balancer {
	return &Balancer{
		pool:     pool,
		strategy: strategy,
	}
}

func (balancer *Balancer) SetStrategy(s strategy.Strategy) {
	balancer.mu.Lock()
	balancer.strategy = s
	balancer.mu.Unlock()
}

func (balancer *Balancer) currentStrategy() strategy.Strategy {
	balancer.mu.RLock()
	defer balancer.mu.RUnlock()

	return balancer.strategy
}

func (balancer *Balancer) Next() (target.Target, error) {
	return balancer.currentStrategy().Select(
		balancer.pool.Healthy(),
	)
}

func (balancer *Balancer) SelectForRequest(key string, excluded map[string]struct{}) (Selection, error) {
	s := balancer.currentStrategy()
	healthy := balancer.pool.Healthy()
	candidates := healthy

	if len(excluded) > 0 {
		filtered := make([]target.Target, 0, len(healthy))
		for _, t := range healthy {
			if _, ok := excluded[t.ID]; !ok {
				filtered = append(filtered, t)
			}
		}
		if len(filtered) > 0 {
			candidates = filtered
		}
	}

	var selected target.Target
	var err error

	if keyed, ok := s.(interface {
		SelectWithKey([]target.Target, string) (target.Target, error)
	}); ok {
		selected, err = keyed.SelectWithKey(candidates, key)
	} else {
		selected, err = s.Select(candidates)
	}

	if err != nil {
		return Selection{}, err
	}

	return Selection{Target: selected, strategy: s}, nil
}

func (selection Selection) Before() {
	if tracker, ok := selection.strategy.(interface{ Inc(string) }); ok {
		tracker.Inc(selection.Target.ID)
	}
}

func (selection Selection) After(d time.Duration) {
	if tracker, ok := selection.strategy.(interface{ Done(string) }); ok {
		tracker.Done(selection.Target.ID)
	}
	if tracker, ok := selection.strategy.(interface {
		Observe(string, time.Duration)
	}); ok {
		tracker.Observe(selection.Target.ID, d)
	}
}

func (balancer *Balancer) ReportFailure(ID string) {
	_ = balancer.pool.RecordFailure(ID)
}

func (balancer *Balancer) ReportSuccess(ID string) {
	_ = balancer.pool.RecordSuccess(ID)
}
