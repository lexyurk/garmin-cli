package main

import (
	"errors"
	"fmt"
	"os"

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

	c, err := client.New(cfgDir, opts.Profile, client.Options{})
	if err == nil {
		return c, nil
	}
	if errors.Is(err, auth.ErrNotAuthenticated) {
		if opts.Format == "json" {
			_ = output.JSONTo(os.Stderr, map[string]any{
				"error":   "not_authenticated",
				"message": "Run `garmin auth login`",
				"profile": orDefault(opts.Profile, "default"),
			})
		} else {
			_ = renderKVTo(os.Stderr, opts.Format, "Authentication", map[string]string{
				"status":  "not authenticated",
				"message": "Run `garmin auth login`",
				"profile": orDefault(opts.Profile, "default"),
			})
		}
		return nil, renderedError(err)
	}

	return nil, fmt.Errorf("init client: %w", err)
}

