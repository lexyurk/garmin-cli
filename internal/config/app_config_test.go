package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAppConfig_MissingFileIsEmpty(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadAppConfig(dir)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if cfg.Format != "" || cfg.Profile != "" {
		t.Fatalf("expected empty config, got: %#v", cfg)
	}
}

func TestLoadAppConfig_ParsesTOML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(path, []byte("format = \"json\"\nprofile = \"work\"\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := LoadAppConfig(dir)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if cfg.Format != "json" || cfg.Profile != "work" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}
