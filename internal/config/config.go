package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
	goyaml "gopkg.in/yaml.v3"
)

type Config struct {
	Domain string `koanf:"domain" yaml:"domain"`
	Token  string `koanf:"token" yaml:"token"`
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.yaml"
	}
	return filepath.Join(home, ".config", "tp", "config.yaml")
}

func Load(path string) (*Config, error) {
	k := koanf.New(".")

	if path == "" {
		path = DefaultPath()
	}

	if _, err := os.Stat(path); err == nil {
		if err := k.Load(file.Provider(path), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("loading config file: %w", err)
		}
	}

	// Environment variables override file config (TP_DOMAIN, TP_TOKEN)
	if err := k.Load(env.Provider("TP_", ".", func(s string) string {
		return strings.ToLower(strings.TrimPrefix(s, "TP_"))
	}), nil); err != nil {
		return nil, fmt.Errorf("loading env config: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.Domain == "" {
		return fmt.Errorf("domain is required (set TP_DOMAIN env var or domain in %s)", DefaultPath())
	}
	if c.Token == "" {
		return fmt.Errorf("token is required (set TP_TOKEN env var or token in %s)", DefaultPath())
	}
	return nil
}

func Get(path, key string) (string, error) {
	cfg, err := Load(path)
	if err != nil {
		return "", err
	}
	switch key {
	case "domain":
		return cfg.Domain, nil
	case "token":
		return cfg.Token, nil
	default:
		return "", fmt.Errorf("unknown config key: %s (valid keys: domain, token)", key)
	}
}

func Set(path, key, value string) error {
	if path == "" {
		path = DefaultPath()
	}
	cfg, err := Load(path)
	if err != nil {
		cfg = &Config{}
	}
	switch key {
	case "domain":
		cfg.Domain = value
	case "token":
		cfg.Token = value
	default:
		return fmt.Errorf("unknown config key: %s (valid keys: domain, token)", key)
	}
	return Save(path, cfg)
}

func Save(path string, cfg *Config) error {
	if path == "" {
		path = DefaultPath()
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := goyaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}
