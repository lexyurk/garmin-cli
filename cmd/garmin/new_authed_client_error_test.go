package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestNewAuthedClient_WrapsInitErrors(t *testing.T) {
	dir := t.TempDir()
	tokensDir := filepath.Join(dir, "tokens", "default")
	if err := os.MkdirAll(tokensDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// Invalid JSON triggers a non-ErrNotAuthenticated error path.
	if err := os.WriteFile(filepath.Join(tokensDir, "oauth1_token.json"), []byte("{not json"), 0o600); err != nil {
		t.Fatalf("write oauth1: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tokensDir, "oauth2_token.json"), []byte(`{"token_type":"bearer","access_token":"ok","expires_at":9999999999}`), 0o600); err != nil {
		t.Fatalf("write oauth2: %v", err)
	}

	opts := &globalOptions{ConfigDir: dir, Profile: ""}
	cmd := &cobra.Command{}
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	c, err := newAuthedClient(cmd, opts)
	if err == nil || c != nil {
		t.Fatalf("expected error and nil client, got client=%v err=%v", c, err)
	}
	if !strings.Contains(err.Error(), "init client") {
		t.Fatalf("expected wrapped init error, got: %v", err)
	}
}
