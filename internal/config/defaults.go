package config

import "time"

const (
	DefaultServerAddress = ":42069"
	DefaultStrategy      = "round_robin"
	DefaultTargetWeight  = 1
	DefaultLogLevel      = "info"
	DefaultLogFormat     = "text"
	DefaultHealthMode    = "tcp"
	DefaultHealthPath    = "/health"
	DefaultRetryAttempts = 1
)

func (cfg *Config) ApplyDefaults() {
	if cfg.Server.Address == "" {
		cfg.Server.Address = DefaultServerAddress
	}

	if cfg.Server.ReadHeaderTimeout <= 0 {
		cfg.Server.ReadHeaderTimeout = 5 * time.Second
	}

	if cfg.Server.ReadTimeout <= 0 {
		cfg.Server.ReadTimeout = 10 * time.Second
	}

	if cfg.Server.WriteTimeout <= 0 {
		cfg.Server.WriteTimeout = 10 * time.Second
	}

	if cfg.Server.IdleTimeout <= 0 {
		cfg.Server.IdleTimeout = 60 * time.Second
	}

	if cfg.Server.ShutdownTimeout <= 0 {
		cfg.Server.ShutdownTimeout = 5 * time.Second
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

	if cfg.Logging.Format == "" {
		cfg.Logging.Format = DefaultLogFormat
	}

	if cfg.Health.Mode == "" {
		cfg.Health.Mode = DefaultHealthMode
	}

	if cfg.Health.Path == "" {
		cfg.Health.Path = DefaultHealthPath
	}

	if cfg.RetryAttempts <= 0 {
		cfg.RetryAttempts = DefaultRetryAttempts
	}
}
