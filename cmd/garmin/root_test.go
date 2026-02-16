package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoot_ConfigTomlDefaultsApplied(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("format = \"json\"\nprofile = \"work\"\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := NewRootCmd("dev")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config-dir", dir, "auth", "status"})

	_ = cmd.Execute()

	got := out.String()
	if !strings.Contains(got, "\"authenticated\": false") {
		t.Fatalf("expected json output, got:\n%s", got)
	}
	if !strings.Contains(got, "\"profile\": \"work\"") {
		t.Fatalf("expected profile from config, got:\n%s", got)
	}
}

func TestRoot_EnvOverridesConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("format = \"markdown\"\nprofile = \"work\"\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	t.Setenv("GARMIN_FORMAT", "json")
	t.Setenv("GARMIN_PROFILE", "envprof")

	cmd := NewRootCmd("dev")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config-dir", dir, "auth", "status"})

	_ = cmd.Execute()

	got := out.String()
	if !strings.Contains(got, "\"authenticated\": false") || !strings.Contains(got, "\"profile\": \"envprof\"") {
		t.Fatalf("expected env overrides, got:\n%s", got)
	}
}

func TestRoot_FlagsOverrideEnv(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("format = \"markdown\"\nprofile = \"work\"\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	t.Setenv("GARMIN_FORMAT", "table")
	t.Setenv("GARMIN_PROFILE", "envprof")

	cmd := NewRootCmd("dev")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config-dir", dir, "--format", "json", "--profile", "flagprof", "auth", "status"})

	_ = cmd.Execute()

	got := out.String()
	if !strings.Contains(got, "\"authenticated\": false") || !strings.Contains(got, "\"profile\": \"flagprof\"") {
		t.Fatalf("expected flags override env, got:\n%s", got)
	}
}

