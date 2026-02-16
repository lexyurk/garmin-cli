package main

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/lexyurk/garmin-cli/internal/client"
	"github.com/lexyurk/garmin-cli/internal/config"
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

			ctx := context.Background()
			results := make([]sleepSummary, 0, len(dates))
			for _, day := range dates {
				var resp sleepDailyResponse
				q := url.Values{"date": {day}}
				if err := c.GetJSON(ctx, "/sleep-service/sleep/dailySleepData", q, &resp); err != nil {
					return err
				}
				results = append(results, resp.toSummary(day))
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

	cmd := &cobra.Command{
		Use:   "heart-rate",
		Short: "Heart rate data",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = opts
			_ = date
			fmt.Println("TODO: garmin health heart-rate")
			return nil
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	return cmd
}

func newHealthStepsCmd(opts *globalOptions) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "steps",
		Short: "Step count",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = opts
			_ = date
			fmt.Println("TODO: garmin health steps")
			return nil
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	return cmd
}

func newHealthStressCmd(opts *globalOptions) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "stress",
		Short: "Stress levels",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = opts
			_ = date
			fmt.Println("TODO: garmin health stress")
			return nil
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	return cmd
}

func newHealthBodyBatteryCmd(opts *globalOptions) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "body-battery",
		Short: "Body battery",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = opts
			_ = date
			fmt.Println("TODO: garmin health body-battery")
			return nil
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	return cmd
}

type sleepDailyResponse struct {
	DailySleepDTO sleepDailyDTO `json:"dailySleepDTO"`
}

type sleepDailyDTO struct {
	CalendarDate            string       `json:"calendarDate"`
	SleepTimeSeconds        *int         `json:"sleepTimeSeconds"`
	DeepSleepSeconds        *int         `json:"deepSleepSeconds"`
	LightSleepSeconds       *int         `json:"lightSleepSeconds"`
	RemSleepSeconds         *int         `json:"remSleepSeconds"`
	AwakeSleepSeconds       *int         `json:"awakeSleepSeconds"`
	AverageSpO2Value        *float64     `json:"averageSpO2Value"`
	AverageRespirationValue *float64     `json:"averageRespirationValue"`
	SleepScores             sleepScores  `json:"sleepScores"`
}

type sleepScores struct {
	Overall sleepScoreOverall `json:"overall"`
}

type sleepScoreOverall struct {
	Value        *int   `json:"value"`
	QualifierKey string `json:"qualifierKey"`
}

type sleepSummary struct {
	Date             string   `json:"date"`
	Score            *int     `json:"score,omitempty"`
	TotalSleepSeconds *int    `json:"total_sleep_seconds,omitempty"`
	DeepSeconds      *int     `json:"deep_seconds,omitempty"`
	LightSeconds     *int     `json:"light_seconds,omitempty"`
	RemSeconds       *int     `json:"rem_seconds,omitempty"`
	AwakeSeconds     *int     `json:"awake_seconds,omitempty"`
	AvgSpO2          *float64 `json:"avg_spo2,omitempty"`
	AvgRespiration   *float64 `json:"avg_respiration,omitempty"`
}

func (r sleepDailyResponse) toSummary(fallbackDate string) sleepSummary {
	date := r.DailySleepDTO.CalendarDate
	if date == "" {
		date = fallbackDate
	}
	return sleepSummary{
		Date:              date,
		Score:             r.DailySleepDTO.SleepScores.Overall.Value,
		TotalSleepSeconds: r.DailySleepDTO.SleepTimeSeconds,
		DeepSeconds:       r.DailySleepDTO.DeepSleepSeconds,
		LightSeconds:      r.DailySleepDTO.LightSleepSeconds,
		RemSeconds:        r.DailySleepDTO.RemSleepSeconds,
		AwakeSeconds:      r.DailySleepDTO.AwakeSleepSeconds,
		AvgSpO2:           r.DailySleepDTO.AverageSpO2Value,
		AvgRespiration:    r.DailySleepDTO.AverageRespirationValue,
	}
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

