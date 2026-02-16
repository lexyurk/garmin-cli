package main

import (
	"context"
	"fmt"
	"time"

	"github.com/lexyurk/garmin-cli/internal/client"
	"github.com/lexyurk/garmin-cli/internal/config"
	"github.com/lexyurk/garmin-cli/internal/output"
	garmintraining "github.com/lexyurk/garmin-cli/internal/training"
	"github.com/lexyurk/garmin-cli/internal/timeutil"
	"github.com/spf13/cobra"
)

func NewTrainingCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "training",
		Short: "Training metrics (status, VO2max, HRV, readiness)",
	}

	cmd.AddCommand(
		newTrainingStatusCmd(opts),
		newTrainingReadinessCmd(opts),
		newTrainingVo2maxCmd(opts),
		newTrainingHrvCmd(opts),
	)

	return cmd
}

func newTrainingStatusCmd(opts *globalOptions) *cobra.Command {
	var date string
	var from string
	var to string
	var days int

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Training status (Productive, Peaking, etc.)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgDir, err := config.ResolveConfigDir(opts.ConfigDir)
			if err != nil {
				return err
			}
			c, err := client.New(cfgDir, opts.Profile, client.Options{})
			if err != nil {
				return err
			}

			dates, err := timeutil.ResolveDates(timeutil.RangeOptions{Date: date, From: from, To: to, Days: days}, time.Now())
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			results, err := mapDatesConcurrently(ctx, dates, 4, func(ctx context.Context, date string) (garmintraining.StatusSummary, error) {
				return garmintraining.GetStatus(ctx, c, date)
			})
			if err != nil {
				return err
			}

			if opts.Format == "json" {
				return output.JSON(results)
			}
			if len(results) == 1 {
				r := results[0]
				return renderKV(opts.Format, "Training status", map[string]string{
					"date":      r.Date,
					"status":    r.StatusPhrase,
					"status_id": formatMaybeInt(r.StatusID),
					"weekly_load": formatMaybeFloatPtr(r.WeeklyTrainingLoad, 0),
					"trend":     r.LoadLevelTrend,
				})
			}
			rows := make([][]string, 0, len(results))
			for _, r := range results {
				rows = append(rows, []string{
					r.Date,
					r.StatusPhrase,
					formatMaybeInt(r.StatusID),
					formatMaybeFloatPtr(r.WeeklyTrainingLoad, 0),
					r.LoadLevelTrend,
				})
			}
			return renderTable(opts.Format, []string{"date", "status", "status_id", "weekly_load", "trend"}, rows)
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD, inclusive)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD, inclusive)")
	cmd.Flags().IntVar(&days, "days", 0, "Shortcut: last N days (ending today)")
	return cmd
}

func newTrainingReadinessCmd(opts *globalOptions) *cobra.Command {
	var date string
	var from string
	var to string
	var days int

	cmd := &cobra.Command{
		Use:   "readiness",
		Short: "Training readiness score (0-100)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgDir, err := config.ResolveConfigDir(opts.ConfigDir)
			if err != nil {
				return err
			}
			c, err := client.New(cfgDir, opts.Profile, client.Options{})
			if err != nil {
				return err
			}

			dates, err := timeutil.ResolveDates(timeutil.RangeOptions{Date: date, From: from, To: to, Days: days}, time.Now())
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			results, err := mapDatesConcurrently(ctx, dates, 4, func(ctx context.Context, date string) (garmintraining.ReadinessSummary, error) {
				return garmintraining.GetReadiness(ctx, c, date)
			})
			if err != nil {
				return err
			}

			if opts.Format == "json" {
				return output.JSON(results)
			}
			if len(results) == 1 {
				r := results[0]
				return renderKV(opts.Format, "Training readiness", map[string]string{
					"date":            r.Date,
					"score":           formatMaybeInt(r.Score),
					"level":           r.Level,
					"sleep_score":     formatMaybeInt(r.SleepScore),
					"hrv_weekly_avg":  formatMaybeInt(r.HRVWeeklyAverage),
					"acute_load":      formatMaybeInt(r.AcuteLoad),
					"recovery_time_s": formatMaybeInt(r.RecoveryTime),
				})
			}

			rows := make([][]string, 0, len(results))
			for _, r := range results {
				rows = append(rows, []string{
					r.Date,
					formatMaybeInt(r.Score),
					r.Level,
					formatMaybeInt(r.SleepScore),
					formatMaybeInt(r.HRVWeeklyAverage),
					formatMaybeInt(r.AcuteLoad),
				})
			}
			return renderTable(opts.Format, []string{"date", "score", "level", "sleep_score", "hrv_weekly_avg", "acute_load"}, rows)
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD, inclusive)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD, inclusive)")
	cmd.Flags().IntVar(&days, "days", 0, "Shortcut: last N days (ending today)")
	return cmd
}

func newTrainingVo2maxCmd(opts *globalOptions) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "vo2max",
		Short: "VO2 max estimates",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = date // kept for backwards-compat; VO2 max is not daily.

			cfgDir, err := config.ResolveConfigDir(opts.ConfigDir)
			if err != nil {
				return err
			}
			c, err := client.New(cfgDir, opts.Profile, client.Options{})
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			vo2, err := garmintraining.GetVO2Max(ctx, c)
			if err != nil {
				return err
			}

			if opts.Format == "json" {
				return output.JSON(map[string]any{
					"running": vo2.Running,
					"cycling": vo2.Cycling,
				})
			}
			return renderKV(opts.Format, "VO2 max", map[string]string{
				"running": formatMaybeFloat0(vo2.Running, 1),
				"cycling": formatMaybeFloat0(vo2.Cycling, 1),
			})
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	_ = cmd.Flags().MarkDeprecated("date", "VO2 max is not daily; this flag will be removed")
	return cmd
}

func newTrainingHrvCmd(opts *globalOptions) *cobra.Command {
	var date string
	var from string
	var to string
	var days int

	cmd := &cobra.Command{
		Use:   "hrv",
		Short: "Heart rate variability",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgDir, err := config.ResolveConfigDir(opts.ConfigDir)
			if err != nil {
				return err
			}
			c, err := client.New(cfgDir, opts.Profile, client.Options{})
			if err != nil {
				return err
			}

			dates, err := timeutil.ResolveDates(timeutil.RangeOptions{Date: date, From: from, To: to, Days: days}, time.Now())
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			results, err := mapDatesConcurrently(ctx, dates, 4, func(ctx context.Context, date string) (garmintraining.HRVSummary, error) {
				return garmintraining.GetHRV(ctx, c, date)
			})
			if err != nil {
				return err
			}

			if opts.Format == "json" {
				return output.JSON(results)
			}
			if len(results) == 1 {
				r := results[0]
				return renderKV(opts.Format, "HRV", map[string]string{
					"date":          r.Date,
					"status":        r.Status,
					"weekly_avg":    formatMaybeInt(r.WeeklyAvg),
					"last_night_avg": formatMaybeInt(r.LastNightAvg),
					"baseline_low":  formatMaybeFloatPtr(r.BaselineLowUpper, 0),
					"baseline_high": formatMaybeFloatPtr(r.BaselineBalancedUpper, 0),
				})
			}
			rows := make([][]string, 0, len(results))
			for _, r := range results {
				rows = append(rows, []string{
					r.Date,
					r.Status,
					formatMaybeInt(r.WeeklyAvg),
					formatMaybeInt(r.LastNightAvg),
				})
			}
			return renderTable(opts.Format, []string{"date", "status", "weekly_avg", "last_night_avg"}, rows)
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD, inclusive)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD, inclusive)")
	cmd.Flags().IntVar(&days, "days", 0, "Shortcut: last N days (ending today)")
	return cmd
}

func formatMaybeFloatPtr(v *float64, decimals int) string {
	if v == nil || *v == 0 {
		return "—"
	}
	format := fmt.Sprintf("%%.%df", decimals)
	return fmt.Sprintf(format, *v)
}

