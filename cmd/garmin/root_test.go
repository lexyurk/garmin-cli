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

func TestRoot_CompletionWorksWithBrokenConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("this is not valid toml\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := NewRootCmd("dev")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config-dir", dir, "completion", "bash"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected completion to work with broken config, got: %v\noutput:\n%s", err, out.String())
	}
	if out.Len() == 0 {
		t.Fatalf("expected completion output, got empty")
	}
}

func TestRoot_VersionWorksWithBrokenConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("this is not valid toml\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := NewRootCmd("dev")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config-dir", dir, "version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected version to work with broken config, got: %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "garmin dev") {
		t.Fatalf("expected version output, got:\n%s", out.String())
	}
}

func TestRoot_HelpCmdWorksWithBrokenConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("this is not valid toml\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := NewRootCmd("dev")
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config-dir", dir, "help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected help to work with broken config, got: %v\noutput:\n%s", err, out.String())
	}
	if !strings.Contains(out.String(), "Fast, ergonomic Garmin Connect CLI") {
		t.Fatalf("expected root help output, got:\n%s", out.String())
	}
}
