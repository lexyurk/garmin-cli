package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lexyurk/garmin-cli/internal/client"
	"github.com/lexyurk/garmin-cli/internal/config"
	garminhealth "github.com/lexyurk/garmin-cli/internal/health"
	"github.com/lexyurk/garmin-cli/internal/output"
	"github.com/lexyurk/garmin-cli/internal/timeutil"
	"github.com/spf13/cobra"
)

func NewHealthCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Health data (sleep, HR, steps, stress, body battery)",
	}

	cmd.AddCommand(
		newHealthSleepCmd(opts),
		newHealthHeartRateCmd(opts),
		newHealthStepsCmd(opts),
		newHealthStressCmd(opts),
		newHealthBodyBatteryCmd(opts),
	)

	return cmd
}

func newHealthSleepCmd(opts *globalOptions) *cobra.Command {
	var date string
	var from string
	var to string
	var days int

	cmd := &cobra.Command{
		Use:   "sleep",
		Short: "Sleep data",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgDir, err := config.ResolveConfigDir(opts.ConfigDir)
			if err != nil {
				return err
			}

			dates, err := timeutil.ResolveDates(timeutil.RangeOptions{
				Date: date,
				From: from,
				To:   to,
				Days: days,
			}, time.Now())
			if err != nil {
				return err
			}

			c, err := client.New(cfgDir, opts.Profile, client.Options{})
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			results, err := mapDatesConcurrently(ctx, dates, 4, func(ctx context.Context, date string) (garminhealth.SleepSummary, error) {
				return garminhealth.GetSleep(ctx, c, date)
			})
			if err != nil {
				return err
			}

			if opts.Format == "json" {
				return output.JSON(results)
			}

			if len(results) == 1 {
				r := results[0]
				return output.MarkdownKV("Sleep", map[string]string{
					"date":       r.Date,
					"score":      formatMaybeInt(r.Score),
					"total":      formatDurationSeconds(r.TotalSleepSeconds),
					"deep":       formatDurationSeconds(r.DeepSeconds),
					"light":      formatDurationSeconds(r.LightSeconds),
					"rem":        formatDurationSeconds(r.RemSeconds),
					"awake":      formatDurationSeconds(r.AwakeSeconds),
					"avg_spo2":   formatMaybeFloat(r.AvgSpO2, 0),
					"avg_breath": formatMaybeFloat(r.AvgRespiration, 1),
				})
			}

			rows := make([][]string, 0, len(results))
			for _, r := range results {
				rows = append(rows, []string{
					r.Date,
					formatMaybeInt(r.Score),
					formatDurationSeconds(r.TotalSleepSeconds),
					formatDurationSeconds(r.DeepSeconds),
					formatDurationSeconds(r.LightSeconds),
					formatDurationSeconds(r.RemSeconds),
					formatDurationSeconds(r.AwakeSeconds),
				})
			}
			return output.MarkdownTable(
				[]string{"date", "score", "total", "deep", "light", "rem", "awake"},
				rows,
			)
		},
	}

	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD, inclusive)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD, inclusive)")
	cmd.Flags().IntVar(&days, "days", 0, "Shortcut: last N days (ending today)")

	return cmd
}

func newHealthHeartRateCmd(opts *globalOptions) *cobra.Command {
	var date string
	var from string
	var to string
	var days int

	cmd := &cobra.Command{
		Use:   "heart-rate",
		Short: "Heart rate data",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgDir, err := config.ResolveConfigDir(opts.ConfigDir)
			if err != nil {
				return err
			}
			dates, err := timeutil.ResolveDates(timeutil.RangeOptions{Date: date, From: from, To: to, Days: days}, time.Now())
			if err != nil {
				return err
			}
			c, err := client.New(cfgDir, opts.Profile, client.Options{})
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			summaries, err := mapDatesConcurrently(ctx, dates, 4, func(ctx context.Context, date string) (garminhealth.DailySummary, error) {
				return garminhealth.GetDailySummary(ctx, c, date)
			})
			if err != nil {
				return err
			}
			out := make([]heartRateSummary, 0, len(summaries))
			for i, s := range summaries {
				d := dates[i]
				out = append(out, heartRateSummary{Date: s.CalendarDateOr(d), RestingHR: s.RestingHeartRate, MinHR: s.MinHeartRate, MaxHR: s.MaxHeartRate})
			}

			if opts.Format == "json" {
				return output.JSON(out)
			}
			if len(out) == 1 {
				r := out[0]
				return output.MarkdownKV("Heart rate", map[string]string{
					"date":        r.Date,
					"resting_hr":  formatMaybeInt(r.RestingHR),
					"min_hr":      formatMaybeInt(r.MinHR),
					"max_hr":      formatMaybeInt(r.MaxHR),
				})
			}
			rows := make([][]string, 0, len(out))
			for _, r := range out {
				rows = append(rows, []string{
					r.Date,
					formatMaybeInt(r.RestingHR),
					formatMaybeInt(r.MinHR),
					formatMaybeInt(r.MaxHR),
				})
			}
			return output.MarkdownTable([]string{"date", "resting_hr", "min_hr", "max_hr"}, rows)
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD, inclusive)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD, inclusive)")
	cmd.Flags().IntVar(&days, "days", 0, "Shortcut: last N days (ending today)")
	return cmd
}

func newHealthStepsCmd(opts *globalOptions) *cobra.Command {
	var date string
	var from string
	var to string
	var days int

	cmd := &cobra.Command{
		Use:   "steps",
		Short: "Step count",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgDir, err := config.ResolveConfigDir(opts.ConfigDir)
			if err != nil {
				return err
			}
			dates, err := timeutil.ResolveDates(timeutil.RangeOptions{Date: date, From: from, To: to, Days: days}, time.Now())
			if err != nil {
				return err
			}
			c, err := client.New(cfgDir, opts.Profile, client.Options{})
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			summaries, err := mapDatesConcurrently(ctx, dates, 4, func(ctx context.Context, date string) (garminhealth.DailySummary, error) {
				return garminhealth.GetDailySummary(ctx, c, date)
			})
			if err != nil {
				return err
			}
			out := make([]stepsSummary, 0, len(summaries))
			for i, s := range summaries {
				d := dates[i]
				out = append(out, stepsSummary{Date: s.CalendarDateOr(d), TotalSteps: s.TotalSteps, Goal: s.DailyStepGoal, DistanceMeters: s.TotalDistanceMeters})
			}

			if opts.Format == "json" {
				return output.JSON(out)
			}
			if len(out) == 1 {
				r := out[0]
				return output.MarkdownKV("Steps", map[string]string{
					"date":     r.Date,
					"steps":    formatMaybeInt(r.TotalSteps),
					"goal":     formatMaybeInt(r.Goal),
					"distance": formatMetersToKM(r.DistanceMeters),
				})
			}
			rows := make([][]string, 0, len(out))
			for _, r := range out {
				rows = append(rows, []string{
					r.Date,
					formatMaybeInt(r.TotalSteps),
					formatMaybeInt(r.Goal),
					formatMetersToKM(r.DistanceMeters),
				})
			}
			return output.MarkdownTable([]string{"date", "steps", "goal", "distance_km"}, rows)
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD, inclusive)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD, inclusive)")
	cmd.Flags().IntVar(&days, "days", 0, "Shortcut: last N days (ending today)")
	return cmd
}

func newHealthStressCmd(opts *globalOptions) *cobra.Command {
	var date string
	var from string
	var to string
	var days int

	cmd := &cobra.Command{
		Use:   "stress",
		Short: "Stress levels",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgDir, err := config.ResolveConfigDir(opts.ConfigDir)
			if err != nil {
				return err
			}
			dates, err := timeutil.ResolveDates(timeutil.RangeOptions{Date: date, From: from, To: to, Days: days}, time.Now())
			if err != nil {
				return err
			}
			c, err := client.New(cfgDir, opts.Profile, client.Options{})
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			summaries, err := mapDatesConcurrently(ctx, dates, 4, func(ctx context.Context, date string) (garminhealth.DailySummary, error) {
				return garminhealth.GetDailySummary(ctx, c, date)
			})
			if err != nil {
				return err
			}
			out := make([]stressSummary, 0, len(summaries))
			for i, s := range summaries {
				d := dates[i]
				out = append(out, stressSummary{Date: s.CalendarDateOr(d), Average: s.AverageStressLevel, Max: s.MaxStressLevel, Qualifier: s.StressQualifier})
			}

			if opts.Format == "json" {
				return output.JSON(out)
			}
			if len(out) == 1 {
				r := out[0]
				return output.MarkdownKV("Stress", map[string]string{
					"date":      r.Date,
					"average":   formatMaybeInt(r.Average),
					"max":       formatMaybeInt(r.Max),
					"qualifier": orDash(r.Qualifier),
				})
			}
			rows := make([][]string, 0, len(out))
			for _, r := range out {
				rows = append(rows, []string{
					r.Date,
					formatMaybeInt(r.Average),
					formatMaybeInt(r.Max),
					orDash(r.Qualifier),
				})
			}
			return output.MarkdownTable([]string{"date", "avg", "max", "qualifier"}, rows)
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD, inclusive)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD, inclusive)")
	cmd.Flags().IntVar(&days, "days", 0, "Shortcut: last N days (ending today)")
	return cmd
}

func newHealthBodyBatteryCmd(opts *globalOptions) *cobra.Command {
	var date string
	var from string
	var to string
	var days int

	cmd := &cobra.Command{
		Use:   "body-battery",
		Short: "Body battery",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfgDir, err := config.ResolveConfigDir(opts.ConfigDir)
			if err != nil {
				return err
			}
			dates, err := timeutil.ResolveDates(timeutil.RangeOptions{Date: date, From: from, To: to, Days: days}, time.Now())
			if err != nil {
				return err
			}
			c, err := client.New(cfgDir, opts.Profile, client.Options{})
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			summaries, err := mapDatesConcurrently(ctx, dates, 4, func(ctx context.Context, date string) (garminhealth.DailySummary, error) {
				return garminhealth.GetDailySummary(ctx, c, date)
			})
			if err != nil {
				return err
			}
			out := make([]bodyBatterySummary, 0, len(summaries))
			for i, s := range summaries {
				d := dates[i]
				out = append(out, bodyBatterySummary{
					Date:       s.CalendarDateOr(d),
					Highest:    s.BodyBatteryHighestValue,
					Lowest:     s.BodyBatteryLowestValue,
					MostRecent: s.BodyBatteryMostRecentValue,
					Charged:    s.BodyBatteryChargedValue,
					Drained:    s.BodyBatteryDrainedValue,
				})
			}

			if opts.Format == "json" {
				return output.JSON(out)
			}
			if len(out) == 1 {
				r := out[0]
				return output.MarkdownKV("Body battery", map[string]string{
					"date":       r.Date,
					"highest":    formatMaybeInt(r.Highest),
					"lowest":     formatMaybeInt(r.Lowest),
					"most_recent": formatMaybeInt(r.MostRecent),
					"charged":    formatMaybeInt(r.Charged),
					"drained":    formatMaybeInt(r.Drained),
				})
			}
			rows := make([][]string, 0, len(out))
			for _, r := range out {
				rows = append(rows, []string{
					r.Date,
					formatMaybeInt(r.Highest),
					formatMaybeInt(r.Lowest),
					formatMaybeInt(r.MostRecent),
				})
			}
			return output.MarkdownTable([]string{"date", "high", "low", "most_recent"}, rows)
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD, inclusive)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD, inclusive)")
	cmd.Flags().IntVar(&days, "days", 0, "Shortcut: last N days (ending today)")
	return cmd
}

func formatDurationSeconds(sec *int) string {
	if sec == nil {
		return "—"
	}
	d := time.Duration(*sec) * time.Second
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func formatMaybeInt(v *int) string {
	if v == nil {
		return "—"
	}
	return fmt.Sprintf("%d", *v)
}

func formatMaybeFloat(v *float64, decimals int) string {
	if v == nil {
		return "—"
	}
	format := fmt.Sprintf("%%.%df", decimals)
	return fmt.Sprintf(format, *v)
}

type stepsSummary struct {
	Date           string `json:"date"`
	TotalSteps     *int   `json:"total_steps,omitempty"`
	Goal           *int   `json:"goal,omitempty"`
	DistanceMeters *int   `json:"distance_meters,omitempty"`
}

type stressSummary struct {
	Date      string `json:"date"`
	Average   *int   `json:"average,omitempty"`
	Max       *int   `json:"max,omitempty"`
	Qualifier string `json:"qualifier,omitempty"`
}

type bodyBatterySummary struct {
	Date       string `json:"date"`
	Highest    *int   `json:"highest,omitempty"`
	Lowest     *int   `json:"lowest,omitempty"`
	MostRecent *int   `json:"most_recent,omitempty"`
	Charged    *int   `json:"charged,omitempty"`
	Drained    *int   `json:"drained,omitempty"`
}

type heartRateSummary struct {
	Date      string `json:"date"`
	RestingHR *int   `json:"resting_hr,omitempty"`
	MinHR     *int   `json:"min_hr,omitempty"`
	MaxHR     *int   `json:"max_hr,omitempty"`
}

func formatMetersToKM(m *int) string {
	if m == nil {
		return "—"
	}
	return fmt.Sprintf("%.2f", float64(*m)/1000.0)
}

func orDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "—"
	}
	return s
}

