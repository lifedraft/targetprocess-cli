package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func cleanKeyring(t *testing.T) {
	t.Helper()
	if err := keyringDelete(); err != nil {
		t.Logf("keyring cleanup skipped: %v", err)
	}
}

func TestResolveTokenSource_EnvWins(t *testing.T) {
	t.Setenv("TP_TOKEN", "env-token")

	cfg := &Config{Token: "file-token"}
	src := resolveTokenSource(cfg)

	if src != TokenSourceEnv {
		t.Errorf("expected TokenSourceEnv, got %s", src)
	}
}

func TestResolveTokenSource_FileToken(t *testing.T) {
	t.Setenv("TP_TOKEN", "")
	cleanKeyring(t)

	cfg := &Config{Token: "file-token"}
	src := resolveTokenSource(cfg)

	if src != TokenSourceFile {
		t.Errorf("expected TokenSourceFile, got %s", src)
	}
}

func TestResolveTokenSource_None(t *testing.T) {
	t.Setenv("TP_TOKEN", "")
	cleanKeyring(t)

	cfg := &Config{}
	src := resolveTokenSource(cfg)

	if src != TokenSourceNone {
		t.Errorf("expected TokenSourceNone, got %s", src)
	}
}

func TestSetToken_FallsBackToFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	if err := os.WriteFile(path, []byte("domain: test.tpondemand.com\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	source, err := SetToken(path, "my-secret-token")
	if err != nil {
		t.Fatalf("SetToken failed: %v", err)
	}

	t.Cleanup(func() { cleanKeyring(t) })

	if source != TokenSourceKeyring && source != TokenSourceFile {
		t.Fatalf("unexpected source: %s", source)
	}

	t.Setenv("TP_TOKEN", "")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if cfg.Token != "my-secret-token" {
		t.Errorf("expected token 'my-secret-token', got %q", cfg.Token)
	}
}

func TestSave_OmitsEmptyToken(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{Domain: "test.tpondemand.com"}
	if err := Save(path, cfg); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(data), "token:") {
		t.Errorf("expected no token field in config file, got:\n%s", data)
	}
}
