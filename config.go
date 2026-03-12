package targetprocess

import (
	"fmt"

	"github.com/lifedraft/targetprocess-cli/internal/config"
)

// Config holds Targetprocess connection settings.
type Config struct {
	Domain string
	Token  string
}

// DefaultConfigPath returns the default config file path (~/.config/tp/config.yaml).
func DefaultConfigPath() string {
	return config.DefaultPath()
}

// LoadConfig loads configuration from the given path (or default if empty).
// Environment variables TP_DOMAIN and TP_TOKEN override file values.
// The OS keyring is also consulted for the token.
func LoadConfig(path string) (*Config, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	return &Config{
		Domain: cfg.Domain,
		Token:  cfg.Token,
	}, nil
}

// NewClientFromConfig creates a Client using configuration from disk/environment.
// It loads config from the default path, consulting env vars and keyring.
func NewClientFromConfig(opts ...Option) (*Client, error) {
	cfg, err := LoadConfig("")
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	if cfg.Domain == "" || cfg.Token == "" {
		return nil, fmt.Errorf("domain and token are required; set TP_DOMAIN/TP_TOKEN env vars or configure %s", DefaultConfigPath())
	}
	return NewClient(cfg.Domain, cfg.Token, opts...)
}
