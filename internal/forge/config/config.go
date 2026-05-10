package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultAgentImage is the default Docker image for the claude-forge agent.
	DefaultAgentImage = "ghcr.io/michael-freling/claude-forge-agent:latest"

	// DefaultGatewayImage is the default Docker image for the claude-forge gateway.
	DefaultGatewayImage = "ghcr.io/michael-freling/claude-forge-gateway:latest"
)

// Config holds the claude-forge configuration.
type Config struct {
	Images   ImagesConfig   `yaml:"images"`
	Defaults DefaultsConfig `yaml:"defaults"`
}

// ImagesConfig holds Docker image configuration.
type ImagesConfig struct {
	Agent   string `yaml:"agent"`
	Gateway string `yaml:"gateway"`
}

// DefaultsConfig holds default behavior configuration.
type DefaultsConfig struct {
	SkipPermissions bool `yaml:"skip_permissions"`
	Worktree        bool `yaml:"worktree"`
}

// DefaultConfig returns a Config with all defaults applied.
func DefaultConfig() *Config {
	return &Config{
		Images: ImagesConfig{
			Agent:   DefaultAgentImage,
			Gateway: DefaultGatewayImage,
		},
	}
}

// Load reads config from configDir/config.yaml.
// Returns default config if file doesn't exist.
// configDir parameter allows overriding the config directory for testing.
func Load(configDir string) (*Config, error) {
	configPath := filepath.Join(configDir, "config.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := DefaultConfig()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Fill in defaults for empty values
	if cfg.Images.Agent == "" {
		cfg.Images.Agent = DefaultAgentImage
	}
	if cfg.Images.Gateway == "" {
		cfg.Images.Gateway = DefaultGatewayImage
	}

	return cfg, nil
}
