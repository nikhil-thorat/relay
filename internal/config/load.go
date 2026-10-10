package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Config, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("load config %s: %w", path, err)
	}

	var cfg Config

	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, fmt.Errorf("load config %s: %w", path, err)
	}

	cfg.ApplyDefaults()

	err = cfg.Validate()
	if err != nil {
		return nil, fmt.Errorf("load config %s: %w", path, err)
	}

	return &cfg, nil

}
