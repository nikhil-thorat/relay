package config

import (
	"errors"
	"fmt"
	"net"
	"strconv"
)

func (cfg *Config) Validate() error {

	if len(cfg.Targets) == 0 {
		return errors.New("no targets defined for relay")
	}

	if cfg.Server.Address == "" {
		return errors.New("server address must not be empty")
	}

	if cfg.Metrics.Enabled && cfg.Metrics.Address == "" {
		return errors.New("metrics address must not be empty when metrics enabled")
	}

	if cfg.Health.Enabled {
		if cfg.Health.Interval <= 0 {
			return errors.New("health interval must be positive when health enabled")
		}
		if cfg.Health.Timeout <= 0 {
			return errors.New("health timeout must be positive when health enabled")
		}
		switch cfg.Health.Mode {
		case "", "tcp", "http":
		default:
			return errors.New("health mode must be tcp or http")
		}
	}

	seen := make(map[string]bool)

	for _, target := range cfg.Targets {
		if target.ID == "" || target.Address == "" {
			return errors.New("invalid target definition, ID or Address missing")
		}

		host, port, err := net.SplitHostPort(target.Address)
		if err != nil || host == "" || port == "" {
			return fmt.Errorf("invalid target address %q", target.Address)
		}

		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return fmt.Errorf("invalid target port in %q", target.Address)
		}

		if seen[target.ID] {
			return errors.New("duplicate target id")
		}

		seen[target.ID] = true
	}

	switch cfg.Strategy.Type {
	case "round_robin", "weighted_round_robin", "least_conn", "least_response_time", "consistent_hash":
	default:
		return errors.New("invalid strategy type")
	}

	switch cfg.Logging.Format {
	case "", "text", "json":
	default:
		return errors.New("logging format must be text or json")
	}

	switch cfg.Logging.Level {
	case "", "debug", "info", "warn", "error", "off":
	default:
		return errors.New("invalid log level")
	}

	if cfg.Server.MaxInflight < 0 {
		return errors.New("max_inflight must not be negative")
	}

	if cfg.Server.RateLimitRPS < 0 {
		return errors.New("rate_limit_rps must not be negative")
	}

	if cfg.RetryAttempts != nil && *cfg.RetryAttempts < 0 {
		return errors.New("retry_attempts must not be negative")
	}

	return nil
}
