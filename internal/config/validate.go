package config

import (
	"errors"
)

func (cfg *Config) Validate() error {

	if len(cfg.Targets) == 0 {
		return errors.New("no targets defined for relay")
	}

	seen := make(map[string]bool)

	for _, target := range cfg.Targets {
		if target.ID == "" || target.Address == "" {
			return errors.New("invalid target definition, ID or Address missing")
		}

		if seen[target.ID] {
			return errors.New("duplicate target id")
		}

		seen[target.ID] = true
	}

	switch cfg.Strategy.Type {
	case "round_robin":
	default:
		return errors.New("invalid strategy type")
	}

	return nil
}
