package relay

import (
	"context"
	"fmt"
	"strings"

	"github.com/nikhil-thorat/relay/internal/config"
	"github.com/nikhil-thorat/relay/internal/health"
	"github.com/nikhil-thorat/relay/internal/strategy"
	"github.com/nikhil-thorat/relay/internal/target"
)

func (relay *Relay) Reload(path string) error {
	cfg, err := config.Load(path)
	if err != nil {
		return fmt.Errorf("reload config %s: %w", path, err)
	}

	relay.lifecycleMu.Lock()
	defer relay.lifecycleMu.Unlock()

	if relay.pool == nil || relay.Balancer == nil {
		return fmt.Errorf("reload: pool not initialized")
	}

	var nextStrategy strategy.Strategy
	if cfg.Strategy.Type != relay.strategyType {
		nextStrategy, err = strategy.New(cfg.Strategy.Type)
		if err != nil {
			return err
		}
	}

	previous := relay.cfg
	healthChanged := previous.Health != cfg.Health

	if healthChanged {
		relay.stopHealth()
	}

	relay.pool.SetPassiveRecovery(!cfg.Health.Enabled)

	wanted := make(map[string]struct{}, len(cfg.Targets))
	order := make([]string, 0, len(cfg.Targets))

	for _, t := range cfg.Targets {
		wanted[t.ID] = struct{}{}
		order = append(order, t.ID)

		next := target.Target{
			ID:      t.ID,
			Address: t.Address,
			Weight:  t.Weight,
		}

		existing, err := relay.pool.Get(t.ID)
		if err != nil {
			_ = relay.pool.Add(&next)
			continue
		}

		if existing.Address != next.Address || existing.Weight != next.Weight {
			_ = relay.pool.Update(&next)
		}
	}

	for _, existing := range relay.pool.List() {
		if _, ok := wanted[existing.ID]; !ok {
			_ = relay.pool.Remove(existing.ID)
		}
	}

	relay.pool.Reorder(order)

	if !cfg.Health.Enabled {
		for _, t := range relay.pool.List() {
			_ = relay.pool.SetHealthy(t.ID, true)
		}
	}

	if relay.Metrics != nil {
		relay.Metrics.SetHealthyTargets(relay.pool.HealthyCount())
	}

	if nextStrategy != nil {
		relay.Balancer.SetStrategy(nextStrategy)
		relay.strategyType = cfg.Strategy.Type
	}

	relay.Balancer.PruneTargets()

	if relay.proxy != nil {
		if previous.Server.MaxInflight != cfg.Server.MaxInflight || previous.Server.RateLimitRPS != cfg.Server.RateLimitRPS {
			relay.proxy.SetLimits(cfg.Server.MaxInflight, cfg.Server.RateLimitRPS)
		}
		if previous.RetryCount() != cfg.RetryCount() {
			relay.proxy.SetRetryAttempts(cfg.RetryCount())
		}
	}

	if healthChanged {
		relay.rebuildHealth(cfg.Health)
	}

	relay.warnRestartRequired(previous, *cfg)
	relay.cfg = *cfg

	if relay.Logger != nil {
		relay.Logger.Info("reloaded config", "path", path, "targets", len(cfg.Targets))
	}

	return nil
}

func (relay *Relay) stopHealth() {
	if relay.healthCancel != nil {
		relay.healthCancel()
	}
	if relay.Health != nil {
		relay.Health.Wait()
	}
}

func (relay *Relay) rebuildHealth(next config.HealthConfig) {
	logger := relay.Logger
	if logger != nil {
		logger = logger.WithComponent("health")
	}

	relay.Health = health.New(
		relay.pool,
		relay.Metrics,
		next.Interval,
		next.Timeout,
		next.Mode,
		next.Path,
		logger,
	)
	relay.healthEnabled = next.Enabled
	relay.healthCtx, relay.healthCancel = context.WithCancel(relay.ctx)

	if relay.started && next.Enabled {
		relay.Health.Start(relay.healthCtx)
	}
}

func (relay *Relay) warnRestartRequired(previous, next config.Config) {
	if relay.Logger == nil {
		return
	}

	fields := make([]string, 0)

	if previous.Server.Address != next.Server.Address {
		fields = append(fields, "server.address")
	}
	if previous.Server.ReadHeaderTimeout != next.Server.ReadHeaderTimeout {
		fields = append(fields, "server.read_header_timeout")
	}
	if previous.Server.ReadTimeout != next.Server.ReadTimeout {
		fields = append(fields, "server.read_timeout")
	}
	if previous.Server.WriteTimeout != next.Server.WriteTimeout {
		fields = append(fields, "server.write_timeout")
	}
	if previous.Server.IdleTimeout != next.Server.IdleTimeout {
		fields = append(fields, "server.idle_timeout")
	}
	if previous.Server.ShutdownTimeout != next.Server.ShutdownTimeout {
		fields = append(fields, "server.shutdown_timeout")
	}
	if previous.Metrics.Enabled != next.Metrics.Enabled {
		fields = append(fields, "metrics.enabled")
	}
	if previous.Metrics.Address != next.Metrics.Address {
		fields = append(fields, "metrics.address")
	}
	if previous.Logging.Level != next.Logging.Level {
		fields = append(fields, "logging.level")
	}
	if previous.Logging.Format != next.Logging.Format {
		fields = append(fields, "logging.format")
	}

	if len(fields) > 0 {
		relay.Logger.Warn("config changes require restart", "fields", strings.Join(fields, ", "))
	}
}
