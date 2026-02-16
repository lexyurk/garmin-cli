package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/lexyurk/garmin-cli/internal/auth"
	"github.com/lexyurk/garmin-cli/internal/client"
	"github.com/lexyurk/garmin-cli/internal/config"
	"github.com/lexyurk/garmin-cli/internal/output"
	"github.com/spf13/cobra"
)

func newAuthedClient(cmd *cobra.Command, opts *globalOptions) (*client.Client, error) {
	cfgDir, err := config.ResolveConfigDir(opts.ConfigDir)
	if err != nil {
		return nil, err
	}

	c, err := client.New(cfgDir, opts.Profile, clientOptionsForCmd(cmd, opts))
	if err == nil {
		return c, nil
	}
	if errors.Is(err, auth.ErrNotAuthenticated) {
		_ = renderNotAuthenticatedTo(cmd.ErrOrStderr(), opts.Format, opts.Profile)
		return nil, renderedError(err)
	}

	return nil, fmt.Errorf("init client: %w", err)
}

func clientOptionsForCmd(cmd *cobra.Command, opts *globalOptions) client.Options {
	if opts == nil || !opts.Verbose || opts.Quiet {
		return client.Options{}
	}
	return client.Options{
		Logf: func(format string, args ...any) {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "garmin verbose: "+format+"\n", args...)
		},
	}
}

func handleAuthedErrorTo(w io.Writer, opts *globalOptions, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, auth.ErrNotAuthenticated) {
		_ = renderNotAuthenticatedTo(w, opts.Format, opts.Profile)
		return renderedError(err)
	}
	return err
}

func renderNotAuthenticatedTo(w io.Writer, format, profile string) error {
	profile = orDefault(profile, "default")

	switch format {
	case "json":
		return output.JSONTo(w, notAuthenticatedJSON{
			Error:   "not_authenticated",
			Message: "Run `garmin auth login`",
			Profile: profile,
		})
	default:
		return renderKVTo(w, format, "Authentication", map[string]string{
			"status":  "not authenticated",
			"message": "Run `garmin auth login`",
			"profile": profile,
		})
	}
}

func renderNotAuthenticatedString(format, profile string) string {
	var b bytes.Buffer
	_ = renderNotAuthenticatedTo(&b, format, profile)
	return b.String()
}

type notAuthenticatedJSON struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Profile string `json:"profile"`
}
