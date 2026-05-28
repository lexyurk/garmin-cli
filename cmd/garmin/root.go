package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/lexyurk/garmin-cli/internal/config"
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
		Use:           "garmin",
		Short:         "Garmin Connect from your terminal",
		Long:          "Fast, ergonomic Garmin Connect CLI. Pipe it, script it, automate it.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// Non-runnable parent commands (e.g., `garmin`, `garmin health`) effectively show help.
			// They should not require a valid config file.
			if !cmd.Runnable() {
				return nil
			}

			// Meta commands should always work, even with broken config.
			switch cmd.Name() {
			case "completion", "help", "version":
				return nil
			}

			// Help should always work, even with broken config.
			if help, _ := cmd.Flags().GetBool("help"); help {
				return nil
			}

			if opts.Verbose && opts.Quiet {
				return fmt.Errorf("use either --verbose or --quiet (not both)")
			}

			// Env defaults (no network calls; safe for `--help`).
			formatFlag := cmd.Flags().Changed("format")
			profileFlag := cmd.Flags().Changed("profile")

			formatEnv := strings.TrimSpace(os.Getenv("GARMIN_FORMAT"))
			profileEnv := strings.TrimSpace(os.Getenv("GARMIN_PROFILE"))

			if !formatFlag && formatEnv != "" {
				opts.Format = formatEnv
			}
			if !profileFlag && profileEnv != "" {
				opts.Profile = profileEnv
			}

			// Config file defaults (lowest precedence).
			needFormatFromFile := !formatFlag && formatEnv == ""
			needProfileFromFile := !profileFlag && profileEnv == ""
			if needFormatFromFile || needProfileFromFile {
				cfgDir, err := config.ResolveConfigDir(opts.ConfigDir)
				if err != nil {
					return err
				}
				fileCfg, err := config.LoadAppConfig(cfgDir)
				if err != nil {
					// If the user explicitly points at a config dir, fail loudly.
					if opts.ConfigDir != "" || os.Getenv("GARMIN_CONFIG_DIR") != "" {
						return err
					}
					// Otherwise, keep defaults.
				} else {
					if needFormatFromFile && strings.TrimSpace(fileCfg.Format) != "" {
						opts.Format = fileCfg.Format
					}
					if needProfileFromFile && strings.TrimSpace(fileCfg.Profile) != "" {
						opts.Profile = fileCfg.Profile
					}
				}
			}

			opts.Format = normalizeFormat(opts.Format)
			switch opts.Format {
			case "markdown", "json", "table", "human":
				// ok
			default:
				return fmt.Errorf("unsupported --format %q (supported: markdown, table, human, json)", opts.Format)
			}

			if err := config.ValidateProfile(opts.Profile); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&opts.Format, "format", "f", "markdown", "Output format: markdown, table, human, json")
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
		NewProfileCmd(opts),
		NewGearCmd(opts),
		NewWorkoutsCmd(opts),
		NewCalendarCmd(opts),
		NewWeightCmd(opts),
		NewDevicesCmd(opts),
		NewRecordsCmd(opts),
		NewTodayCmd(opts),
		NewCompletionCmd(),
		NewVersionCmd(version),
	)

	return cmd
}

func normalizeFormat(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	switch s {
	case "", "markdown", "md":
		return "markdown"
	case "table", "tbl":
		return "table"
	case "human":
		return "human"
	case "json":
		return "json"
	default:
		return s
	}
}
