package relay

import (
	"github.com/nikhil-thorat/relay/internal/balancer"
	"github.com/nikhil-thorat/relay/internal/config"
	"github.com/nikhil-thorat/relay/internal/strategy"
	"github.com/nikhil-thorat/relay/internal/target"
)

type Relay struct {
	Balancer *balancer.Balancer
}

func New(cfg *config.Config) (*Relay, error) {
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

	return &Relay{
		Balancer: balancer,
	}, nil
}
