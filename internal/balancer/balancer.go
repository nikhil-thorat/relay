package balancer

import (
	"github.com/nikhil-thorat/relay/internal/strategy"
	"github.com/nikhil-thorat/relay/internal/target"
)

type Balancer struct {
	pool     *target.Pool
	strategy strategy.Strategy
}

func NewBalancer(pool *target.Pool, strategy strategy.Strategy) *Balancer {
	return &Balancer{
		pool:     pool,
		strategy: strategy,
	}
}

func (balancer *Balancer) Next() (*target.Target, error) {
	return balancer.strategy.Select(
		balancer.pool.Healthy(),
	)
}
