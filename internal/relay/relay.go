package relay

import (
	"github.com/nikhil-thorat/relay/internal/balancer"
	"github.com/nikhil-thorat/relay/internal/config"
	"github.com/nikhil-thorat/relay/internal/health"
	"github.com/nikhil-thorat/relay/internal/metrics"
	"github.com/nikhil-thorat/relay/internal/strategy"
	"github.com/nikhil-thorat/relay/internal/target"
	"github.com/prometheus/client_golang/prometheus"
)

type Relay struct {
	Balancer *balancer.Balancer
	Health   *health.Checker
	Metrics  *metrics.Metrics
}

func New(cfg *config.Config, registry prometheus.Registerer) (*Relay, error) {
	pool := target.NewPool()

	for _, t := range cfg.Targets {
		err := pool.Add(&target.Target{
			ID:      t.ID,
			Address: t.Address,
			Weight:  t.Weight,
		})
		if err != nil {
			return nil, err
		}
	}

	strat, err := strategy.New(cfg.Strategy.Type)
	if err != nil {
		return nil, err
	}

	balancer := balancer.New(pool, strat)

	metrics := metrics.New(
		registry,
	)

	checker := health.New(
		pool,
		metrics,
		cfg.Health.Interval,
		cfg.Health.Timeout,
	)

	return &Relay{
		Balancer: balancer,
		Health:   checker,
		Metrics:  metrics,
	}, nil
}
