package config

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v4"
)

// Load loads a config file from the given path and validates it.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var raw any
	err = yaml.Unmarshal(data, &raw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := validate(raw); err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}
