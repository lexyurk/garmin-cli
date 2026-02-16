package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/lexyurk/garmin-cli/internal/client"
	"github.com/lexyurk/garmin-cli/internal/config"
	"github.com/lexyurk/garmin-cli/internal/output"
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

			ctx := context.Background()
			results := make([]trainingStatusSummary, 0, len(dates))
			for _, d := range dates {
				raw := map[string]any{}
				path := fmt.Sprintf("/mobile-gateway/usersummary/trainingstatus/latest/%s", d)
				if err := c.GetJSON(ctx, path, nil, &raw); err != nil {
					return err
				}
				results = append(results, summarizeTrainingStatus(d, raw))
			}

			if opts.Format == "json" {
				return output.JSON(results)
			}
			if len(results) == 1 {
				r := results[0]
				return output.MarkdownKV("Training status", map[string]string{
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
			return output.MarkdownTable([]string{"date", "status", "status_id", "weekly_load", "trend"}, rows)
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

			ctx := context.Background()
			results := make([]trainingReadinessSummary, 0, len(dates))
			for _, d := range dates {
				var entries []trainingReadinessEntry
				if err := c.GetJSON(ctx, fmt.Sprintf("/metrics-service/metrics/trainingreadiness/%s", d), nil, &entries); err != nil {
					return err
				}
				results = append(results, summarizeTrainingReadiness(d, entries))
			}

			if opts.Format == "json" {
				return output.JSON(results)
			}
			if len(results) == 1 {
				r := results[0]
				return output.MarkdownKV("Training readiness", map[string]string{
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
			return output.MarkdownTable(
				[]string{"date", "score", "level", "sleep_score", "hrv_weekly_avg", "acute_load"},
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

func newTrainingVo2maxCmd(opts *globalOptions) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "vo2max",
		Short: "VO2 max estimates",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = opts
			_ = date
			fmt.Println("TODO: garmin training vo2max")
			return nil
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
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

			ctx := context.Background()
			results := make([]hrvSummary, 0, len(dates))
			for _, d := range dates {
				var resp hrvResponse
				if err := c.GetJSON(ctx, fmt.Sprintf("/hrv-service/hrv/%s", d), nil, &resp); err != nil {
					return err
				}
				results = append(results, resp.toSummary(d))
			}

			if opts.Format == "json" {
				return output.JSON(results)
			}
			if len(results) == 1 {
				r := results[0]
				return output.MarkdownKV("HRV", map[string]string{
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
			return output.MarkdownTable([]string{"date", "status", "weekly_avg", "last_night_avg"}, rows)
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD, inclusive)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD, inclusive)")
	cmd.Flags().IntVar(&days, "days", 0, "Shortcut: last N days (ending today)")
	return cmd
}

type trainingReadinessEntry struct {
	CalendarDate      string `json:"calendarDate"`
	Timestamp         string `json:"timestamp"`
	Level             string `json:"level"`
	Score             *int   `json:"score"`
	SleepScore        *int   `json:"sleepScore"`
	HRVWeeklyAverage  *int   `json:"hrvWeeklyAverage"`
	AcuteLoad         *int   `json:"acuteLoad"`
	RecoveryTime      *int   `json:"recoveryTime"`
}

type trainingReadinessSummary struct {
	Date            string `json:"date"`
	Level           string `json:"level,omitempty"`
	Score           *int   `json:"score,omitempty"`
	SleepScore      *int   `json:"sleep_score,omitempty"`
	HRVWeeklyAverage *int  `json:"hrv_weekly_avg,omitempty"`
	AcuteLoad       *int   `json:"acute_load,omitempty"`
	RecoveryTime    *int   `json:"recovery_time_seconds,omitempty"`
}

func summarizeTrainingReadiness(fallbackDate string, entries []trainingReadinessEntry) trainingReadinessSummary {
	best := trainingReadinessEntry{}
	for _, e := range entries {
		if e.Timestamp > best.Timestamp {
			best = e
		}
	}
	date := best.CalendarDate
	if date == "" {
		date = fallbackDate
	}
	return trainingReadinessSummary{
		Date:             date,
		Level:            best.Level,
		Score:            best.Score,
		SleepScore:       best.SleepScore,
		HRVWeeklyAverage: best.HRVWeeklyAverage,
		AcuteLoad:        best.AcuteLoad,
		RecoveryTime:     best.RecoveryTime,
	}
}

type hrvResponse struct {
	HRVSummary hrvSummaryDTO `json:"hrvSummary"`
}

type hrvSummaryDTO struct {
	CalendarDate   string            `json:"calendarDate"`
	WeeklyAvg      *int              `json:"weeklyAvg"`
	LastNightAvg   *int              `json:"lastNightAvg"`
	Status         string            `json:"status"`
	Baseline       map[string]any    `json:"baseline"`
}

type hrvSummary struct {
	Date                  string   `json:"date"`
	WeeklyAvg             *int     `json:"weekly_avg,omitempty"`
	LastNightAvg          *int     `json:"last_night_avg,omitempty"`
	Status                string   `json:"status,omitempty"`
	BaselineLowUpper      *float64 `json:"baseline_low_upper,omitempty"`
	BaselineBalancedUpper *float64 `json:"baseline_balanced_upper,omitempty"`
}

func (r hrvResponse) toSummary(fallbackDate string) hrvSummary {
	date := r.HRVSummary.CalendarDate
	if date == "" {
		date = fallbackDate
	}

	s := hrvSummary{
		Date:         date,
		WeeklyAvg:    r.HRVSummary.WeeklyAvg,
		LastNightAvg: r.HRVSummary.LastNightAvg,
		Status:       r.HRVSummary.Status,
	}
	if v, ok := r.HRVSummary.Baseline["lowUpper"].(float64); ok {
		s.BaselineLowUpper = &v
	}
	if v, ok := r.HRVSummary.Baseline["balancedUpper"].(float64); ok {
		s.BaselineBalancedUpper = &v
	}
	return s
}

type trainingStatusSummary struct {
	Date              string    `json:"date"`
	StatusPhrase      string    `json:"status_phrase,omitempty"`
	StatusID          *int      `json:"status_id,omitempty"`
	WeeklyTrainingLoad *float64 `json:"weekly_training_load,omitempty"`
	LoadLevelTrend    string    `json:"load_level_trend,omitempty"`
}

func summarizeTrainingStatus(date string, raw map[string]any) trainingStatusSummary {
	s := trainingStatusSummary{Date: date}

	// mostRecentTrainingStatus.payload.latestTrainingStatusData.{deviceId} -> dict
	mr, _ := raw["mostRecentTrainingStatus"].(map[string]any)
	payload, _ := mr["payload"].(map[string]any)
	latest, _ := payload["latestTrainingStatusData"].(map[string]any)
	for _, v := range latest {
		if entry, ok := v.(map[string]any); ok {
			if phrase, ok := entry["trainingStatusFeedbackPhrase"].(string); ok {
				s.StatusPhrase = phrase
			}
			if id, ok := entry["trainingStatus"].(float64); ok {
				i := int(id)
				s.StatusID = &i
			}
			if wl, ok := entry["weeklyTrainingLoad"].(float64); ok {
				s.WeeklyTrainingLoad = &wl
			}
			if trend, ok := entry["loadLevelTrend"].(string); ok {
				s.LoadLevelTrend = trend
			}
			break
		}
	}

	if strings.TrimSpace(s.StatusPhrase) == "" && s.StatusID == nil {
		// Garmin returns many permutations; keep it stable.
		s.StatusPhrase = "—"
	}

	return s
}

func formatMaybeFloatPtr(v *float64, decimals int) string {
	if v == nil || *v == 0 {
		return "—"
	}
	format := fmt.Sprintf("%%.%df", decimals)
	return fmt.Sprintf(format, *v)
}

