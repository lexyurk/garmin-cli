package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func renderNotAuthenticatedString(format, profile string) string {
	var b bytes.Buffer
	_ = renderNotAuthenticatedTo(&b, format, profile)
	return b.String()
}

func TestRenderNotAuthenticatedString_Markdown(t *testing.T) {
	got := renderNotAuthenticatedString("markdown", "")
	if !strings.Contains(got, "## Authentication") {
		t.Fatalf("missing title: %q", got)
	}
	if !strings.Contains(got, "not authenticated") {
		t.Fatalf("missing status: %q", got)
	}
}

func TestRenderNotAuthenticatedString_JSON(t *testing.T) {
	got := renderNotAuthenticatedString("json", "")
	if !strings.Contains(got, "\"error\": \"not_authenticated\"") {
		t.Fatalf("missing error json: %q", got)
	}
	if !strings.Contains(got, "\"profile\": \"default\"") {
		t.Fatalf("missing profile json: %q", got)
	}
}

func TestRenderNotAuthenticatedString_Table(t *testing.T) {
	got := renderNotAuthenticatedString("table", "")
	if !strings.Contains(got, "field") || !strings.Contains(got, "value") {
		t.Fatalf("expected table headers, got: %q", got)
	}
	if !strings.Contains(got, "not authenticated") {
		t.Fatalf("expected status in table, got: %q", got)
	}
}

func TestRenderNotAuthenticatedString_Human(t *testing.T) {
	got := renderNotAuthenticatedString("human", "")
	if !strings.Contains(got, "=== Authentication ===") {
		t.Fatalf("expected title, got: %q", got)
	}
	if !strings.Contains(got, "status: not authenticated") {
		t.Fatalf("expected status, got: %q", got)
	}
}

func TestClientOptionsForCmd_VerboseWritesToStderr(t *testing.T) {
	cmd := &cobra.Command{}
	var stderr bytes.Buffer
	cmd.SetErr(&stderr)

	opts := &globalOptions{Verbose: true, Quiet: false}
	copts := clientOptionsForCmd(cmd, opts)
	if copts.Logf == nil {
		t.Fatalf("expected Logf to be set when verbose=true")
	}

	copts.Logf("hello %s", "world")
	if !strings.Contains(stderr.String(), "garmin verbose: hello world") {
		t.Fatalf("expected log to go to stderr, got:\n%s", stderr.String())
	}
}

func TestClientOptionsForCmd_QuietDisablesVerbose(t *testing.T) {
	cmd := &cobra.Command{}
	opts := &globalOptions{Verbose: true, Quiet: true}
	copts := clientOptionsForCmd(cmd, opts)
	if copts.Logf != nil {
		t.Fatalf("expected Logf to be nil when quiet=true")
	}
}
