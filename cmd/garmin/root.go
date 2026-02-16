package main

import (
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

