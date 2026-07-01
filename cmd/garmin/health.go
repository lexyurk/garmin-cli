package main

import (
	"context"
	"fmt"
	"strings"
	"time"

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
		newHealthSummaryCmd(opts),
		newHealthSleepCmd(opts),
		newHealthHeartRateCmd(opts),
		newHealthStepsCmd(opts),
		newHealthStressCmd(opts),
		newHealthBodyBatteryCmd(opts),
		newHealthSpo2Cmd(opts),
		newHealthRespirationCmd(opts),
		newHealthIntensityMinutesCmd(opts),
	)

	return cmd
}

func newHealthSpo2Cmd(opts *globalOptions) *cobra.Command {
	var date, from, to string
	var days int

	cmd := &cobra.Command{
		Use:   "spo2",
		Short: "Pulse ox (SpO2)",
		RunE: func(cmd *cobra.Command, args []string) error {
			dates, err := timeutil.ResolveDates(timeutil.RangeOptions{Date: date, From: from, To: to, Days: days}, time.Now())
			if err != nil {
				return err
			}
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			results, err := mapDatesConcurrently(ctx, dates, 4, func(ctx context.Context, date string) (garminhealth.SpO2Summary, error) {
				return garminhealth.GetSpO2(ctx, c, date)
			})
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), results)
			}
			if len(results) == 1 {
				r := results[0]
				return renderKVTo(cmd.OutOrStdout(), opts.Format, "SpO2", map[string]string{
					"date":    r.Date,
					"average": formatMaybeFloat(r.Average, 0),
					"lowest":  formatMaybeFloat(r.Lowest, 0),
					"latest":  formatMaybeFloat(r.Latest, 0),
				})
			}
			rows := make([][]string, 0, len(results))
			for _, r := range results {
				rows = append(rows, []string{r.Date, formatMaybeFloat(r.Average, 0), formatMaybeFloat(r.Lowest, 0)})
			}
			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"date", "average", "lowest"}, rows)
		},
	}
	addDateRangeFlags(cmd, &date, &from, &to, &days)
	return cmd
}

func newHealthRespirationCmd(opts *globalOptions) *cobra.Command {
	var date, from, to string
	var days int

	cmd := &cobra.Command{
		Use:   "respiration",
		Short: "Respiration rate (breaths/min)",
		RunE: func(cmd *cobra.Command, args []string) error {
			dates, err := timeutil.ResolveDates(timeutil.RangeOptions{Date: date, From: from, To: to, Days: days}, time.Now())
			if err != nil {
				return err
			}
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			results, err := mapDatesConcurrently(ctx, dates, 4, func(ctx context.Context, date string) (garminhealth.RespirationSummary, error) {
				return garminhealth.GetRespiration(ctx, c, date)
			})
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), results)
			}
			if len(results) == 1 {
				r := results[0]
				return renderKVTo(cmd.OutOrStdout(), opts.Format, "Respiration", map[string]string{
					"date":       r.Date,
					"avg_waking": formatMaybeFloat(r.AvgWaking, 0),
					"highest":    formatMaybeFloat(r.Highest, 0),
					"lowest":     formatMaybeFloat(r.Lowest, 0),
				})
			}
			rows := make([][]string, 0, len(results))
			for _, r := range results {
				rows = append(rows, []string{r.Date, formatMaybeFloat(r.AvgWaking, 0), formatMaybeFloat(r.Highest, 0), formatMaybeFloat(r.Lowest, 0)})
			}
			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"date", "avg_waking", "highest", "lowest"}, rows)
		},
	}
	addDateRangeFlags(cmd, &date, &from, &to, &days)
	return cmd
}

func newHealthIntensityMinutesCmd(opts *globalOptions) *cobra.Command {
	var date, from, to string
	var days int

	cmd := &cobra.Command{
		Use:   "intensity-minutes",
		Short: "Intensity minutes (moderate/vigorous)",
		RunE: func(cmd *cobra.Command, args []string) error {
			dates, err := timeutil.ResolveDates(timeutil.RangeOptions{Date: date, From: from, To: to, Days: days}, time.Now())
			if err != nil {
				return err
			}
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			results, err := mapDatesConcurrently(ctx, dates, 4, func(ctx context.Context, date string) (garminhealth.IntensityMinutesSummary, error) {
				return garminhealth.GetIntensityMinutes(ctx, c, date)
			})
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), results)
			}
			if len(results) == 1 {
				r := results[0]
				return renderKVTo(cmd.OutOrStdout(), opts.Format, "Intensity minutes", map[string]string{
					"date":        r.Date,
					"moderate":    formatMaybeFloat(r.Moderate, 0),
					"vigorous":    formatMaybeFloat(r.Vigorous, 0),
					"weekly_goal": formatMaybeFloat(r.WeeklyGoal, 0),
				})
			}
			rows := make([][]string, 0, len(results))
			for _, r := range results {
				rows = append(rows, []string{r.Date, formatMaybeFloat(r.Moderate, 0), formatMaybeFloat(r.Vigorous, 0)})
			}
			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"date", "moderate", "vigorous"}, rows)
		},
	}
	addDateRangeFlags(cmd, &date, &from, &to, &days)
	return cmd
}

func addDateRangeFlags(cmd *cobra.Command, date, from, to *string, days *int) {
	cmd.Flags().StringVar(date, "date", "", "Date (YYYY-MM-DD, default: today)")
	cmd.Flags().StringVar(from, "from", "", "Start date (YYYY-MM-DD, inclusive)")
	cmd.Flags().StringVar(to, "to", "", "End date (YYYY-MM-DD, inclusive)")
	cmd.Flags().IntVar(days, "days", 0, "Shortcut: last N days (ending today)")
}

func newHealthSummaryCmd(opts *globalOptions) *cobra.Command {
	var date string
	var from string
	var to string
	var days int

	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Daily health summary (steps, HR, stress, body battery)",
		RunE: func(cmd *cobra.Command, args []string) error {
			dates, err := timeutil.ResolveDates(timeutil.RangeOptions{Date: date, From: from, To: to, Days: days}, time.Now())
			if err != nil {
				return err
			}
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			summaries, err := mapDatesConcurrently(ctx, dates, 4, func(ctx context.Context, date string) (garminhealth.DailySummary, error) {
				return garminhealth.GetDailySummary(ctx, c, date)
			})
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			out := make([]dailySummaryCompact, 0, len(summaries))
			for i, s := range summaries {
				d := dates[i]
				out = append(out, dailySummaryCompact{
					Date:            s.CalendarDateOr(d),
					Steps:           s.TotalSteps,
					RestingHR:       s.RestingHeartRate,
					StressAvg:       s.AverageStressLevel,
					StressMax:       s.MaxStressLevel,
					StressQualifier: s.StressQualifier,
					BodyBatteryHigh: s.BodyBatteryHighestValue,
					BodyBatteryLow:  s.BodyBatteryLowestValue,
				})
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), out)
			}
			if len(out) == 1 {
				r := out[0]
				return renderKVTo(cmd.OutOrStdout(), opts.Format, "Daily summary", map[string]string{
					"date":              r.Date,
					"steps":             formatMaybeInt(r.Steps),
					"resting_hr":        formatMaybeInt(r.RestingHR),
					"stress_avg":        formatMaybeInt(r.StressAvg),
					"stress_max":        formatMaybeInt(r.StressMax),
					"stress_qualifier":  orDash(r.StressQualifier),
					"body_battery_high": formatMaybeInt(r.BodyBatteryHigh),
					"body_battery_low":  formatMaybeInt(r.BodyBatteryLow),
				})
			}

			rows := make([][]string, 0, len(out))
			for _, r := range out {
				rows = append(rows, []string{
					r.Date,
					formatMaybeInt(r.Steps),
					formatMaybeInt(r.RestingHR),
					formatMaybeInt(r.StressAvg),
					formatMaybeInt(r.BodyBatteryHigh),
					formatMaybeInt(r.BodyBatteryLow),
				})
			}
			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"date", "steps", "resting_hr", "stress_avg", "bb_high", "bb_low"}, rows)
		},
	}

	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD, inclusive)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD, inclusive)")
	cmd.Flags().IntVar(&days, "days", 0, "Shortcut: last N days (ending today)")
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
			dates, err := timeutil.ResolveDates(timeutil.RangeOptions{
				Date: date,
				From: from,
				To:   to,
				Days: days,
			}, time.Now())
			if err != nil {
				return err
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			results, err := mapDatesConcurrently(ctx, dates, 4, func(ctx context.Context, date string) (garminhealth.SleepSummary, error) {
				return garminhealth.GetSleep(ctx, c, date)
			})
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), results)
			}

			if len(results) == 1 {
				r := results[0]
				return renderKVTo(cmd.OutOrStdout(), opts.Format, "Sleep", map[string]string{
					"date":        r.Date,
					"score":       formatMaybeInt(r.Score),
					"total":       formatDurationSeconds(r.TotalSleepSeconds),
					"deep":        formatDurationSeconds(r.DeepSeconds),
					"light":       formatDurationSeconds(r.LightSeconds),
					"rem":         formatDurationSeconds(r.RemSeconds),
					"awake":       formatDurationSeconds(r.AwakeSeconds),
					"avg_spo2":    formatMaybeFloat(r.AvgSpO2, 0),
					"avg_breath":  formatMaybeFloat(r.AvgRespiration, 1),
					"sleep_start": formatMaybeString(r.SleepStart),
					"sleep_end":   formatMaybeString(r.SleepEnd),
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
					formatMaybeString(r.SleepStart),
					formatMaybeString(r.SleepEnd),
				})
			}
			return renderTableTo(
				cmd.OutOrStdout(),
				opts.Format,
				[]string{"date", "score", "total", "deep", "light", "rem", "awake", "sleep_start", "sleep_end"},
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
			dates, err := timeutil.ResolveDates(timeutil.RangeOptions{Date: date, From: from, To: to, Days: days}, time.Now())
			if err != nil {
				return err
			}
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			summaries, err := mapDatesConcurrently(ctx, dates, 4, func(ctx context.Context, date string) (garminhealth.DailySummary, error) {
				return garminhealth.GetDailySummary(ctx, c, date)
			})
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}
			out := make([]heartRateSummary, 0, len(summaries))
			for i, s := range summaries {
				d := dates[i]
				out = append(out, heartRateSummary{Date: s.CalendarDateOr(d), RestingHR: s.RestingHeartRate, MinHR: s.MinHeartRate, MaxHR: s.MaxHeartRate})
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), out)
			}
			if len(out) == 1 {
				r := out[0]
				return renderKVTo(cmd.OutOrStdout(), opts.Format, "Heart rate", map[string]string{
					"date":       r.Date,
					"resting_hr": formatMaybeInt(r.RestingHR),
					"min_hr":     formatMaybeInt(r.MinHR),
					"max_hr":     formatMaybeInt(r.MaxHR),
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
			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"date", "resting_hr", "min_hr", "max_hr"}, rows)
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
			dates, err := timeutil.ResolveDates(timeutil.RangeOptions{Date: date, From: from, To: to, Days: days}, time.Now())
			if err != nil {
				return err
			}
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			summaries, err := mapDatesConcurrently(ctx, dates, 4, func(ctx context.Context, date string) (garminhealth.DailySummary, error) {
				return garminhealth.GetDailySummary(ctx, c, date)
			})
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}
			out := make([]stepsSummary, 0, len(summaries))
			for i, s := range summaries {
				d := dates[i]
				out = append(out, stepsSummary{Date: s.CalendarDateOr(d), TotalSteps: s.TotalSteps, Goal: s.DailyStepGoal, DistanceMeters: s.TotalDistanceMeters})
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), out)
			}
			if len(out) == 1 {
				r := out[0]
				return renderKVTo(cmd.OutOrStdout(), opts.Format, "Steps", map[string]string{
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
			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"date", "steps", "goal", "distance_km"}, rows)
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
	var values bool

	cmd := &cobra.Command{
		Use:   "stress",
		Short: "Stress levels",
		RunE: func(cmd *cobra.Command, args []string) error {
			dates, err := timeutil.ResolveDates(timeutil.RangeOptions{Date: date, From: from, To: to, Days: days}, time.Now())
			if err != nil {
				return err
			}
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			if values {
				details, err := mapDatesConcurrently(ctx, dates, 4, func(ctx context.Context, date string) (garminhealth.StressDetail, error) {
					return garminhealth.GetStressDetail(ctx, c, date)
				})
				if err != nil {
					return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
				}
				// Intraday timelines are data, not prose: always emit JSON.
				return output.JSONTo(cmd.OutOrStdout(), details)
			}
			summaries, err := mapDatesConcurrently(ctx, dates, 4, func(ctx context.Context, date string) (garminhealth.DailySummary, error) {
				return garminhealth.GetDailySummary(ctx, c, date)
			})
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}
			out := make([]stressSummary, 0, len(summaries))
			for i, s := range summaries {
				d := dates[i]
				out = append(out, stressSummary{Date: s.CalendarDateOr(d), Average: s.AverageStressLevel, Max: s.MaxStressLevel, Qualifier: s.StressQualifier})
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), out)
			}
			if len(out) == 1 {
				r := out[0]
				return renderKVTo(cmd.OutOrStdout(), opts.Format, "Stress", map[string]string{
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
			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"date", "avg", "max", "qualifier"}, rows)
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD, inclusive)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD, inclusive)")
	cmd.Flags().IntVar(&days, "days", 0, "Shortcut: last N days (ending today)")
	cmd.Flags().BoolVar(&values, "values", false, "Include intraday timeline (~3-min samples, JSON output; -1 unmeasured, -2 activity)")
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
			dates, err := timeutil.ResolveDates(timeutil.RangeOptions{Date: date, From: from, To: to, Days: days}, time.Now())
			if err != nil {
				return err
			}
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			summaries, err := mapDatesConcurrently(ctx, dates, 4, func(ctx context.Context, date string) (garminhealth.DailySummary, error) {
				return garminhealth.GetDailySummary(ctx, c, date)
			})
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
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
				return output.JSONTo(cmd.OutOrStdout(), out)
			}
			if len(out) == 1 {
				r := out[0]
				return renderKVTo(cmd.OutOrStdout(), opts.Format, "Body battery", map[string]string{
					"date":        r.Date,
					"highest":     formatMaybeInt(r.Highest),
					"lowest":      formatMaybeInt(r.Lowest),
					"most_recent": formatMaybeInt(r.MostRecent),
					"charged":     formatMaybeInt(r.Charged),
					"drained":     formatMaybeInt(r.Drained),
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
			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"date", "high", "low", "most_recent"}, rows)
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

func formatMaybeString(v *string) string {
	if v == nil {
		return "—"
	}
	return orDash(*v)
}

func formatMaybeFloat(v *float64, decimals int) string {
	return formatMaybeFloatWithZeroOption(v, decimals, false)
}

func formatMaybeFloatZeroAsDash(v *float64, decimals int) string {
	return formatMaybeFloatWithZeroOption(v, decimals, true)
}

func formatMaybeFloatWithZeroOption(v *float64, decimals int, zeroAsDash bool) string {
	if v == nil || (zeroAsDash && *v == 0) {
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

type dailySummaryCompact struct {
	Date            string `json:"date"`
	Steps           *int   `json:"steps,omitempty"`
	RestingHR       *int   `json:"resting_hr,omitempty"`
	StressAvg       *int   `json:"stress_avg,omitempty"`
	StressMax       *int   `json:"stress_max,omitempty"`
	StressQualifier string `json:"stress_qualifier,omitempty"`
	BodyBatteryHigh *int   `json:"body_battery_high,omitempty"`
	BodyBatteryLow  *int   `json:"body_battery_low,omitempty"`
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
