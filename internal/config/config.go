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

// TokenSource describes where the token was resolved from.
type TokenSource string

const (
	TokenSourceNone    TokenSource = "none"
	TokenSourceKeyring TokenSource = "keyring"
	TokenSourceEnv     TokenSource = "env"
	TokenSourceFile    TokenSource = "file"
)

const (
	keyDomain = "domain"
	keyToken  = "token"
)

type Config struct {
	Domain string `koanf:"domain" yaml:"domain"`
	Token  string `koanf:"token" yaml:"token"`

	// TokenSource indicates where the token was loaded from (not persisted).
	TokenSource TokenSource `koanf:"-" yaml:"-"`
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

	// Determine token source with priority: env > keyring > file
	cfg.TokenSource = resolveTokenSource(&cfg)

	return &cfg, nil
}

// resolveTokenSource determines where the token came from and fills it from
// the keyring if no higher-priority source provided one.
func resolveTokenSource(cfg *Config) TokenSource {
	// Check if TP_TOKEN env var is set (highest priority).
	if os.Getenv("TP_TOKEN") != "" {
		return TokenSourceEnv
	}

	// Try the OS keyring.
	if token, err := keyringGet(); err == nil && token != "" {
		if cfg.Token == "" {
			cfg.Token = token
		}
		// If the file also had a token, keyring still wins (we already have it).
		// But if user explicitly set TP_TOKEN env, that already returned above.
		if cfg.Token == token {
			return TokenSourceKeyring
		}
	}

	// Token came from the config file.
	if cfg.Token != "" {
		return TokenSourceFile
	}

	return TokenSourceNone
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
	case keyDomain:
		return cfg.Domain, nil
	case keyToken:
		return cfg.Token, nil
	default:
		return "", fmt.Errorf("unknown config key: %s (valid keys: domain, token)", key)
	}
}

// SetToken stores the token using the most secure available backend.
// It tries the OS keyring first; if unavailable, falls back to the config file.
// Returns the storage location used and any error.
func SetToken(path, token string) (TokenSource, error) {
	if err := keyringSet(token); err == nil {
		// Stored in keyring — remove token from the config file if present.
		if err := clearFileToken(path); err != nil {
			return TokenSourceKeyring, fmt.Errorf("stored in keyring but failed to clear file token: %w", err)
		}
		return TokenSourceKeyring, nil
	}

	// Keyring unavailable — fall back to config file.
	return TokenSourceFile, setFileValue(path, keyToken, token)
}

func Set(path, key, value string) error {
	if key == keyToken {
		_, err := SetToken(path, value)
		return err
	}
	return setFileValue(path, key, value)
}

func setFileValue(path, key, value string) error {
	if path == "" {
		path = DefaultPath()
	}
	cfg, err := Load(path)
	if err != nil {
		cfg = &Config{}
	}
	switch key {
	case keyDomain:
		cfg.Domain = value
	case keyToken:
		cfg.Token = value
	default:
		return fmt.Errorf("unknown config key: %s (valid keys: domain, token)", key)
	}
	return Save(path, cfg)
}

// clearFileToken removes the token field from the config file,
// keeping other settings (like domain) intact.
func clearFileToken(path string) error {
	if path == "" {
		path = DefaultPath()
	}
	if _, err := os.Stat(path); err != nil {
		return nil //nolint:nilerr // no file means nothing to clean
	}
	cfg, err := Load(path)
	if err != nil {
		return err
	}
	cfg.Token = ""
	return Save(path, cfg)
}

func Save(path string, cfg *Config) error {
	if path == "" {
		path = DefaultPath()
	}

	// Only persist domain and token to file (strip transient fields).
	fileCfg := struct {
		Domain string `yaml:"domain"`
		Token  string `yaml:"token,omitempty"`
	}{
		Domain: cfg.Domain,
		Token:  cfg.Token,
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}
	data, err := goyaml.Marshal(fileCfg)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}
