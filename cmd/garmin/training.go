package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lexyurk/garmin-cli/internal/output"
	"github.com/lexyurk/garmin-cli/internal/timeutil"
	garmintraining "github.com/lexyurk/garmin-cli/internal/training"
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
		newTrainingFitnessAgeCmd(opts),
	)

	return cmd
}

func newTrainingFitnessAgeCmd(opts *globalOptions) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "fitness-age",
		Short: "Fitness age (a.k.a. health age) estimate",
		RunE: func(cmd *cobra.Command, args []string) error {
			d := strings.TrimSpace(date)
			if d == "" {
				d = time.Now().In(time.Local).Format("2006-01-02")
			} else if _, err := time.ParseInLocation("2006-01-02", d, time.Local); err != nil {
				return fmt.Errorf("invalid --date %q (expected YYYY-MM-DD)", date)
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			fa, err := garmintraining.GetFitnessAge(ctx, c, d)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), fa)
			}

			fields := map[string]string{
				"date":                   fa.Date,
				"fitness_age":            formatMaybeFloat(fa.FitnessAge, 1),
				"chronological_age":      formatMaybeFloat(fa.ChronologicalAge, 1),
				"achievable_fitness_age": formatMaybeFloat(fa.AchievableAge, 1),
				"previous_fitness_age":   formatMaybeFloat(fa.PreviousAge, 1),
			}
			for k, v := range fa.Components {
				fields["component_"+k] = strconv.FormatFloat(v, 'f', -1, 64)
			}
			return renderKVTo(cmd.OutOrStdout(), opts.Format, "Fitness age", fields)
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
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
			c, err := newAuthedClient(cmd, opts)
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
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), results)
			}
			if len(results) == 1 {
				r := results[0]
				return renderKVTo(cmd.OutOrStdout(), opts.Format, "Training status", map[string]string{
					"date":        r.Date,
					"status":      r.StatusPhrase,
					"status_id":   formatMaybeInt(r.StatusID),
					"weekly_load": formatMaybeFloatZeroAsDash(r.WeeklyTrainingLoad, 0),
					"trend":       r.LoadLevelTrend,
				})
			}
			rows := make([][]string, 0, len(results))
			for _, r := range results {
				rows = append(rows, []string{
					r.Date,
					r.StatusPhrase,
					formatMaybeInt(r.StatusID),
					formatMaybeFloatZeroAsDash(r.WeeklyTrainingLoad, 0),
					r.LoadLevelTrend,
				})
			}
			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"date", "status", "status_id", "weekly_load", "trend"}, rows)
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
			c, err := newAuthedClient(cmd, opts)
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
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), results)
			}
			if len(results) == 1 {
				r := results[0]
				return renderKVTo(cmd.OutOrStdout(), opts.Format, "Training readiness", map[string]string{
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
			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"date", "score", "level", "sleep_score", "hrv_weekly_avg", "acute_load"}, rows)
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD, inclusive)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD, inclusive)")
	cmd.Flags().IntVar(&days, "days", 0, "Shortcut: last N days (ending today)")
	return cmd
}

func newTrainingVo2maxCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vo2max",
		Short: "VO2 max estimates",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			vo2, err := garmintraining.GetVO2Max(ctx, c)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), vo2maxJSON{
					Running: vo2.Running,
					Cycling: vo2.Cycling,
				})
			}
			return renderKVTo(cmd.OutOrStdout(), opts.Format, "VO2 max", map[string]string{
				"running": formatMaybeFloat0(vo2.Running, 1),
				"cycling": formatMaybeFloat0(vo2.Cycling, 1),
			})
		},
	}
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
			c, err := newAuthedClient(cmd, opts)
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
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), results)
			}
			if len(results) == 1 {
				r := results[0]
				return renderKVTo(cmd.OutOrStdout(), opts.Format, "HRV", map[string]string{
					"date":           r.Date,
					"status":         r.Status,
					"weekly_avg":     formatMaybeInt(r.WeeklyAvg),
					"last_night_avg": formatMaybeInt(r.LastNightAvg),
					"baseline_low":   formatMaybeFloatZeroAsDash(r.BaselineLowUpper, 0),
					"baseline_high":  formatMaybeFloatZeroAsDash(r.BaselineBalancedUpper, 0),
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
			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"date", "status", "weekly_avg", "last_night_avg"}, rows)
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD, inclusive)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD, inclusive)")
	cmd.Flags().IntVar(&days, "days", 0, "Shortcut: last N days (ending today)")
	return cmd
}

type vo2maxJSON struct {
	Running float64 `json:"running"`
	Cycling float64 `json:"cycling"`
}
