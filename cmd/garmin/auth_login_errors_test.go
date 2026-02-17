package main

import (
	"strings"
	"testing"
)

func TestAuthLogin_PasswordAndPasswordStdinMutuallyExclusive(t *testing.T) {
	opts := &globalOptions{}
	cmd := newAuthLoginCmd(opts)
	cmd.SetArgs([]string{"--email", "user@example.com", "--password", "x", "--password-stdin"})
	cmd.SetIn(strings.NewReader("ignored"))

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "password-stdin") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthLogin_PasswordStdinEmptyReturnsMissingCredentials(t *testing.T) {
	opts := &globalOptions{}
	cmd := newAuthLoginCmd(opts)
	cmd.SetArgs([]string{"--email", "user@example.com", "--password-stdin"})
	cmd.SetIn(strings.NewReader(""))

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "missing credentials") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthLogin_UsesEnvEmail_AndFailsOnBadConfigDir(t *testing.T) {
	t.Setenv("GARMIN_EMAIL", "env@example.com")

	opts := &globalOptions{ConfigDir: "   "}
	cmd := newAuthLoginCmd(opts)
	cmd.SetArgs([]string{"--password", "x"})
	cmd.SetIn(strings.NewReader("ignored"))

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "empty path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAuthLogin_UsesEnvPassword_AndFailsOnBadConfigDir(t *testing.T) {
	t.Setenv("GARMIN_PASSWORD", "env-pass")

	opts := &globalOptions{ConfigDir: "   "}
	cmd := newAuthLoginCmd(opts)
	cmd.SetArgs([]string{"--email", "user@example.com"})
	cmd.SetIn(strings.NewReader("ignored"))

	err := cmd.Execute()
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "empty path") {
		t.Fatalf("unexpected error: %v", err)
	}
}
