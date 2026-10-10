package relay

import (
	"fmt"

	"github.com/nikhil-thorat/relay/internal/config"
	"github.com/nikhil-thorat/relay/internal/strategy"
	"github.com/nikhil-thorat/relay/internal/target"
)

func (relay *Relay) Reload(path string) error {
	cfg, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("reload config %s: %w", path, err)
	}

	if relay.pool == nil || relay.Balancer == nil {
		return fmt.Errorf("reload: pool not initialized")
	}

	want := make(map[string]target.Target)
	for _, t := range cfg.Targets {
		want[t.ID] = target.Target{ID: t.ID, Address: t.Address, Weight: t.Weight}
	}

	for _, existing := range relay.pool.List() {
		next, ok := want[existing.ID]
		if !ok {
			_ = relay.pool.Remove(existing.ID)
			continue
		}
		if existing.Address != next.Address || existing.Weight != next.Weight {
			_ = relay.pool.Update(&next)
		}
		delete(want, existing.ID)
	}

	for _, t := range want {
		_ = relay.pool.Add(&t)
	}

	strat, err := strategy.New(cfg.Strategy.Type)
	if err != nil {
		return err
	}
	relay.Balancer.SetStrategy(strat)

	if relay.proxy != nil {
		relay.proxy.SetLimits(cfg.Server.MaxInflight, cfg.Server.RateLimitRPS)
	}

	if relay.Logger != nil {
		relay.Logger.Info("reloaded config", "path", path, "targets", len(cfg.Targets))
	}

	return nil
}
