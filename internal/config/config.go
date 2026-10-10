package config

import "time"

type ServerConfig struct {
	Address           string        `yaml:"address"`
	ReadHeaderTimeout time.Duration `yaml:"read_header_timeout"`
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	IdleTimeout       time.Duration `yaml:"idle_timeout"`
	ShutdownTimeout   time.Duration `yaml:"shutdown_timeout"`
	MaxInflight       int           `yaml:"max_inflight"`
	RateLimitRPS      int           `yaml:"rate_limit_rps"`
}

type StrategyConfig struct {
	Type string `yaml:"type"`
}

type TargetConfig struct {
	ID      string `yaml:"id"`
	Address string `yaml:"address"`
	Weight  int    `yaml:"weight"`
}

type HealthConfig struct {
	Enabled  bool          `yaml:"enabled"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
	Mode     string        `yaml:"mode"`
	Path     string        `yaml:"path"`
}

type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Address string `yaml:"address"`
}

type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

type Config struct {
	Server        ServerConfig   `yaml:"server"`
	Strategy      StrategyConfig `yaml:"strategy"`
	Targets       []TargetConfig `yaml:"targets"`
	Health        HealthConfig   `yaml:"health"`
	Metrics       MetricsConfig  `yaml:"metrics"`
	Logging       LoggingConfig  `yaml:"logging"`
	RetryAttempts int            `yaml:"retry_attempts"`
}
