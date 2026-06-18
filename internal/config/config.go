package config

import "time"

type ServerConfig struct {
	Address string `yaml:"address"`
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
}

type MetricsConfig struct {
	Enabled bool   `yaml:"enabled"`
	Address string `yaml:"address"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Strategy StrategyConfig `yaml:"strategy"`
	Targets  []TargetConfig `yaml:"targets"`
	Health   HealthConfig   `yaml:"health"`
	Metrics  MetricsConfig  `yaml:"metrics"`
	Logging  LoggingConfig  `yaml:"logging"`
}
