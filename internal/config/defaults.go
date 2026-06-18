package config

const (
	DefaultServerAddress = ":42069"
	DefaultStrategy      = "round_robin"
	DefaultTargetWeight  = 1
	DefaultLogLevel      = "info"
)

func (cfg *Config) ApplyDefaults() {
	if cfg.Server.Address == "" {
		cfg.Server.Address = DefaultServerAddress
	}

	if cfg.Strategy.Type == "" {
		cfg.Strategy.Type = DefaultStrategy
	}

	for idx := range cfg.Targets {
		if cfg.Targets[idx].Weight <= 0 {
			cfg.Targets[idx].Weight = DefaultTargetWeight
		}
	}

	if cfg.Logging.Level == "" {
		cfg.Logging.Level = DefaultLogLevel
	}
}
