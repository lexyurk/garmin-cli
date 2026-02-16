package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNotAuthenticated_GoesToStderr_StdoutClean(t *testing.T) {
	dir := t.TempDir()

	// Ensure config exists (optional), but no token files.
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), []byte("format = \"markdown\"\n"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	cmd := NewRootCmd("dev")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--config-dir", dir, "health", "sleep"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}

	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("expected stdout to be empty, got:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "not authenticated") {
		t.Fatalf("expected stderr to contain not authenticated message, got:\n%s", stderr.String())
	}
}

func TestNotAuthenticated_JSONGoesToStderr_StdoutClean(t *testing.T) {
	dir := t.TempDir()

	cmd := NewRootCmd("dev")
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs([]string{"--config-dir", dir, "--format", "json", "health", "sleep"})

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}

	if strings.TrimSpace(stdout.String()) != "" {
		t.Fatalf("expected stdout to be empty, got:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "\"error\": \"not_authenticated\"") {
		t.Fatalf("expected stderr to contain json not_authenticated, got:\n%s", stderr.String())
	}
}

