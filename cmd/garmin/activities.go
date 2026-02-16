package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	garminactivities "github.com/lexyurk/garmin-cli/internal/activities"
	"github.com/lexyurk/garmin-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewActivitiesCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "activities",
		Short: "Activity management",
	}

	cmd.AddCommand(
		newActivitiesListCmd(opts),
		newActivitiesGetCmd(opts),
		newActivitiesSplitsCmd(opts),
	)

	return cmd
}

func newActivitiesListCmd(opts *globalOptions) *cobra.Command {
	var limit int
	var after string
	var before string
	var activityType string
	var days int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List activities",
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("--limit must be > 0")
			}

			afterResolved, beforeResolved, err := resolveActivitiesDateFilters(after, before, days, time.Now())
			if err != nil {
				return err
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			out, err := garminactivities.List(ctx, c, limit, afterResolved, beforeResolved, activityType)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), out)
			}

			rows := make([][]string, 0, len(out))
			for _, a := range out {
				rows = append(rows, []string{
					fmt.Sprintf("%d", a.ID),
					a.StartTimeLocal,
					a.Type,
					a.Name,
					formatDistanceKM(a.DistanceMeters),
					formatDurationSecondsFloat(a.DurationSeconds),
					formatMaybeInt0(a.Calories),
					formatMaybeInt0(a.AvgHR),
				})
			}

			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"id", "start", "type", "name", "dist_km", "duration", "kcal", "avg_hr"}, rows)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Number of activities to return")
	cmd.Flags().StringVar(&after, "after", "", "Activities after date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&before, "before", "", "Activities before date (YYYY-MM-DD)")
	cmd.Flags().IntVar(&days, "days", 0, "Shortcut: last N days (ending today)")
	cmd.Flags().StringVar(&activityType, "type", "", "Activity type filter (running, cycling, etc.)")

	return cmd
}

func newActivitiesGetCmd(opts *globalOptions) *cobra.Command {
	var details bool

	cmd := &cobra.Command{
		Use:   "get [activity-id]",
		Short: "Get activity details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid activity id %q", args[0])
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			raw, err := garminactivities.GetRaw(ctx, c, id)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), raw)
			}

			s := garminactivities.SummarizeDetail(id, raw)
			fields := map[string]string{
				"id":        fmt.Sprintf("%d", s.ID),
				"date_time": s.StartTimeLocal,
				"type":      s.Type,
				"name":      s.Name,
				"distance":  formatDistanceKM(s.DistanceMeters) + " km",
				"duration":  formatDurationSecondsFloat(s.DurationSeconds),
				"calories":  formatMaybeInt0(s.Calories),
				"avg_hr":    formatMaybeInt0(s.AvgHR),
				"max_hr":    formatMaybeInt0(s.MaxHR),
				"elev_gain": formatMaybeFloat0(s.ElevationGain, 0) + " m",
			}
			if details {
				// Keep it small and stable; callers can use --format json for full detail.
				fields["vo2max"] = formatMaybeFloat0(s.VO2Max, 1)
				fields["training_load"] = formatMaybeFloat0(s.TrainingLoad, 0)
			}

			return renderKVTo(cmd.OutOrStdout(), opts.Format, "Activity", fields)
		},
	}

	cmd.Flags().BoolVar(&details, "details", false, "Include extended details")
	return cmd
}

func newActivitiesSplitsCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "splits [activity-id]",
		Short: "Get activity splits/laps",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid activity id %q", args[0])
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			raw, err := garminactivities.GetRaw(ctx, c, id)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), activitiesSplitsJSON{
					ActivityID: id,
					Splits:     raw["splitSummaries"],
				})
			}

			splits := garminactivities.ExtractSplits(raw)
			rows := make([][]string, 0, len(splits))
			for i, s := range splits {
				rows = append(rows, []string{
					fmt.Sprintf("%d", i+1),
					formatDistanceKM(s.DistanceMeters),
					formatDurationSecondsFloat(s.DurationSeconds),
					formatPaceMinPerKM(s.DistanceMeters, s.DurationSeconds),
					formatMaybeInt0(s.AverageHR),
					formatMaybeInt0(s.MaxHR),
				})
			}

			if len(rows) == 0 {
				return renderKVTo(cmd.OutOrStdout(), opts.Format, "Splits", map[string]string{
					"activity_id": fmt.Sprintf("%d", id),
					"message":     "No splits available",
				})
			}

			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"split", "dist_km", "duration", "pace_min_per_km", "avg_hr", "max_hr"}, rows)
		},
	}
	return cmd
}

func formatDistanceKM(m float64) string {
	if m <= 0 {
		return "—"
	}
	return fmt.Sprintf("%.2f", m/1000.0)
}

func formatDurationSecondsFloat(sec float64) string {
	if sec <= 0 {
		return "—"
	}
	d := time.Duration(sec * float64(time.Second))
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm %02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm %02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

func formatMaybeInt0(v int) string {
	if v == 0 {
		return "—"
	}
	return fmt.Sprintf("%d", v)
}

func formatMaybeFloat0(v float64, decimals int) string {
	if v == 0 {
		return "—"
	}
	format := fmt.Sprintf("%%.%df", decimals)
	return fmt.Sprintf(format, v)
}

func formatPaceMinPerKM(distanceMeters, durationSeconds float64) string {
	if distanceMeters <= 0 || durationSeconds <= 0 {
		return "—"
	}
	km := distanceMeters / 1000.0
	secPerKM := durationSeconds / km
	d := time.Duration(secPerKM * float64(time.Second))
	min := int(d.Minutes())
	sec := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", min, sec)
}

type activitiesSplitsJSON struct {
	ActivityID int64 `json:"activity_id"`
	Splits     any   `json:"splits"`
}

func resolveActivitiesDateFilters(after, before string, days int, now time.Time) (string, string, error) {
	if days < 0 {
		return "", "", fmt.Errorf("--days must be >= 0")
	}
	if days > 0 && (strings.TrimSpace(after) != "" || strings.TrimSpace(before) != "") {
		return "", "", fmt.Errorf("use either --days or --after/--before (not both)")
	}
	if days == 0 {
		return after, before, nil
	}

	end := now.In(time.Local).Format("2006-01-02")
	start := now.In(time.Local).AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	return start, end, nil
}
