package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

type globalOptions struct {
	Format    string
	Verbose   bool
	Quiet     bool
	ConfigDir string
	Profile   string
}

func NewRootCmd(version string) *cobra.Command {
	opts := &globalOptions{}

	cmd := &cobra.Command{
		Use:   "garmin",
		Short: "Garmin Connect from your terminal",
		Long:  "Fast, ergonomic Garmin Connect CLI. Pipe it, script it, automate it.",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Env defaults (no network calls; safe for `--help`).
			if !cmd.Flags().Changed("format") {
				if env := strings.TrimSpace(os.Getenv("GARMIN_FORMAT")); env != "" {
					opts.Format = env
				}
			}
			if !cmd.Flags().Changed("profile") {
				if env := strings.TrimSpace(os.Getenv("GARMIN_PROFILE")); env != "" {
					opts.Profile = env
				}
			}

			opts.Format = normalizeFormat(opts.Format)
			if opts.Format != "markdown" && opts.Format != "json" {
				return fmt.Errorf("unsupported --format %q (supported: markdown, json)", opts.Format)
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&opts.Format, "format", "f", "markdown", "Output format: markdown, json")
	cmd.PersistentFlags().BoolVarP(&opts.Verbose, "verbose", "v", false, "Verbose output")
	cmd.PersistentFlags().BoolVarP(&opts.Quiet, "quiet", "q", false, "Suppress non-essential output")
	cmd.PersistentFlags().StringVarP(&opts.ConfigDir, "config-dir", "c", "", "Config directory (default: ~/.config/garmin)")
	cmd.PersistentFlags().StringVar(&opts.ConfigDir, "config", "", "Deprecated: use --config-dir")
	_ = cmd.PersistentFlags().MarkDeprecated("config", "use --config-dir")
	cmd.PersistentFlags().StringVarP(&opts.Profile, "profile", "p", "", "Named profile to use")

	cmd.AddCommand(
		NewAuthCmd(opts),
		NewHealthCmd(opts),
		NewActivitiesCmd(opts),
		NewTrainingCmd(opts),
		NewVersionCmd(version),
	)

	return cmd
}

func normalizeFormat(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "", "markdown", "md":
		return "markdown"
	case "json":
		return "json"
	default:
		return s
	}
}

