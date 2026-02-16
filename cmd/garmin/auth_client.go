package main

import (
	"errors"
	"fmt"

	"github.com/lexyurk/garmin-cli/internal/auth"
	"github.com/lexyurk/garmin-cli/internal/client"
	"github.com/lexyurk/garmin-cli/internal/config"
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
		_ = renderKV(opts.Format, "Authentication", map[string]string{
			"status":  "not authenticated",
			"message": "Run `garmin auth login`",
			"profile": orDefault(opts.Profile, "default"),
		})
		return nil, renderedError(err)
	}

	return nil, fmt.Errorf("init client: %w", err)
}

