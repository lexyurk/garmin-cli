package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lexyurk/garmin-cli/internal/output"
	"github.com/lexyurk/garmin-cli/internal/weight"
	"github.com/spf13/cobra"
)

func NewWeightCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "weight",
		Short: "Weight and body composition",
	}

	cmd.AddCommand(
		newWeightListCmd(opts),
		newWeightLatestCmd(opts),
		newWeightLogCmd(opts),
	)

	return cmd
}

func newWeightListCmd(opts *globalOptions) *cobra.Command {
	var from string
	var to string
	var days int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List weigh-ins over a date range",
		RunE: func(cmd *cobra.Command, args []string) error {
			start, end, err := resolveWeightRange(from, to, days, time.Now())
			if err != nil {
				return err
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			out, err := weight.List(ctx, c, start, end)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), out)
			}
			rows := make([][]string, 0, len(out))
			for _, w := range out {
				rows = append(rows, []string{
					w.Date,
					formatMaybeFloat(w.WeightKG, 1),
					formatMaybeFloat(w.BMI, 1),
					formatMaybeFloat(w.BodyFatPct, 1),
				})
			}
			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"date", "weight_kg", "bmi", "body_fat_%"}, rows)
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD)")
	cmd.Flags().IntVar(&days, "days", 0, "Shortcut: last N days (ending today; default 30)")
	return cmd
}

func newWeightLatestCmd(opts *globalOptions) *cobra.Command {
	var days int

	cmd := &cobra.Command{
		Use:   "latest",
		Short: "Show the most recent weigh-in",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			w, err := weight.Latest(ctx, c, days)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), w)
			}
			return renderKVTo(cmd.OutOrStdout(), opts.Format, "Latest weigh-in", map[string]string{
				"date":       orDash(w.Date),
				"weight_kg":  formatMaybeFloat(w.WeightKG, 1),
				"bmi":        formatMaybeFloat(w.BMI, 1),
				"body_fat_%": formatMaybeFloat(w.BodyFatPct, 1),
			})
		},
	}

	cmd.Flags().IntVar(&days, "days", 30, "Look back this many days for the latest weigh-in")
	return cmd
}

func newWeightLogCmd(opts *globalOptions) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "log [kg]",
		Short: "Log a weigh-in (kilograms)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			kg, err := strconv.ParseFloat(strings.TrimSpace(args[0]), 64)
			if err != nil {
				return fmt.Errorf("invalid weight %q (expected kilograms, e.g. 74.5)", args[0])
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			if err := weight.Add(ctx, c, kg, date); err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			when := strings.TrimSpace(date)
			if when == "" {
				when = "now"
			}
			return renderKVTo(cmd.OutOrStdout(), opts.Format, "Weigh-in logged", map[string]string{
				"weight_kg": strconv.FormatFloat(kg, 'f', -1, 64),
				"date":      when,
				"status":    "logged",
			})
		},
	}

	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	return cmd
}

func resolveWeightRange(from, to string, days int, now time.Time) (string, string, error) {
	if days < 0 {
		return "", "", fmt.Errorf("--days must be >= 0")
	}
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	if days > 0 && (from != "" || to != "") {
		return "", "", fmt.Errorf("use either --days or --from/--to (not both)")
	}

	loc := now.In(time.Local)
	if from == "" && to == "" {
		n := days
		if n == 0 {
			n = 30
		}
		end := loc.Format("2006-01-02")
		start := loc.AddDate(0, 0, -(n - 1)).Format("2006-01-02")
		return start, end, nil
	}
	if to == "" {
		to = loc.Format("2006-01-02")
	}
	if from == "" {
		from = loc.AddDate(0, 0, -29).Format("2006-01-02")
	}
	return from, to, nil
}
