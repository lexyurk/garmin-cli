package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/lexyurk/garmin-cli/internal/auth"
	"github.com/spf13/cobra"
)

func TestHandleAuthedErrorTo_RendersNotAuthenticated(t *testing.T) {
	var b bytes.Buffer
	opts := &globalOptions{Format: "markdown", Profile: ""}

	err := handleAuthedErrorTo(&b, opts, auth.ErrNotAuthenticated)
	if err == nil {
		t.Fatalf("expected error")
	}
	var ce *cliError
	if !errors.As(err, &ce) || !ce.rendered {
		t.Fatalf("expected rendered cliError, got: %T %#v", err, err)
	}
	if !strings.Contains(b.String(), "not authenticated") {
		t.Fatalf("expected output to mention not authenticated, got:\n%s", b.String())
	}
}

func TestCliError_ErrorMethod(t *testing.T) {
	e := renderedError(errors.New("boom"))
	ce, ok := e.(*cliError)
	if !ok {
		t.Fatalf("expected *cliError, got %T", e)
	}
	if ce.Error() != "boom" {
		t.Fatalf("unexpected Error(): %q", ce.Error())
	}
}

func TestClientOptionsForCmd_BaseURLFromEnv(t *testing.T) {
	t.Setenv("GARMIN_CONNECTAPI_BASE_URL", "http://example.test")
	cmd := &cobra.Command{}
	opts := &globalOptions{Verbose: false, Quiet: false}

	copts := clientOptionsForCmd(cmd, opts)
	if copts.BaseURL != "http://example.test" {
		t.Fatalf("expected BaseURL from env, got %q", copts.BaseURL)
	}
}
