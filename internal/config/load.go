package config

import (
	"bytes"
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

	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	err = decoder.Decode(&cfg)
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
