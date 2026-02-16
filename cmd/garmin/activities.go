package main

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lexyurk/garmin-cli/internal/client"
	"github.com/lexyurk/garmin-cli/internal/config"
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

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List activities",
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("--limit must be > 0")
			}

			cfgDir, err := config.ResolveConfigDir(opts.ConfigDir)
			if err != nil {
				return err
			}
			c, err := client.New(cfgDir, opts.Profile, client.Options{})
			if err != nil {
				return err
			}

			ctx := context.Background()

			var out []activitySummary
			start := 0
			pageSize := 50
			if limit < pageSize {
				pageSize = limit
			}

			for len(out) < limit {
				var page []activityListItem
				q := url.Values{
					"limit": {strconv.Itoa(pageSize)},
					"start": {strconv.Itoa(start)},
				}
				if err := c.GetJSON(ctx, "/activitylist-service/activities/search/activities", q, &page); err != nil {
					return err
				}
				if len(page) == 0 {
					break
				}

				for _, item := range page {
					s := item.toSummary()
					if !passesActivityFilters(s, after, before, activityType) {
						continue
					}
					out = append(out, s)
					if len(out) >= limit {
						break
					}
				}

				// Stop early when we paged beyond the requested date range.
				if after != "" {
					oldest := page[len(page)-1].startDate()
					if oldest != "" && oldest < after {
						break
					}
				}

				start += len(page)
				if len(page) < pageSize {
					break
				}
			}

			if opts.Format == "json" {
				return output.JSON(out)
			}

			rows := make([][]string, 0, len(out))
			for _, a := range out {
				rows = append(rows, []string{
					fmt.Sprintf("%d", a.ID),
					a.Date,
					a.Type,
					a.Name,
					formatDistanceKM(a.DistanceMeters),
					formatDurationSecondsFloat(a.DurationSeconds),
					formatMaybeInt0(a.Calories),
					formatMaybeInt0(a.AvgHR),
				})
			}

			return output.MarkdownTable(
				[]string{"id", "date", "type", "name", "dist_km", "duration", "kcal", "avg_hr"},
				rows,
			)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Number of activities to return")
	cmd.Flags().StringVar(&after, "after", "", "Activities after date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&before, "before", "", "Activities before date (YYYY-MM-DD)")
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

			cfgDir, err := config.ResolveConfigDir(opts.ConfigDir)
			if err != nil {
				return err
			}
			c, err := client.New(cfgDir, opts.Profile, client.Options{})
			if err != nil {
				return err
			}

			ctx := context.Background()

			var raw map[string]any
			if err := c.GetJSON(ctx, fmt.Sprintf("/activity-service/activity/%d", id), nil, &raw); err != nil {
				return err
			}

			if opts.Format == "json" {
				return output.JSON(raw)
			}

			s := summarizeActivityDetail(id, raw)
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

			return output.MarkdownKV("Activity", fields)
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
			_ = opts
			fmt.Printf("TODO: garmin activities splits %s\n", args[0])
			return nil
		},
	}
	return cmd
}

type activityTypeInfo struct {
	TypeKey string `json:"typeKey"`
}

type activityListItem struct {
	ActivityID     int64            `json:"activityId"`
	ActivityName   string           `json:"activityName"`
	StartTimeLocal string           `json:"startTimeLocal"`
	ActivityType   activityTypeInfo `json:"activityType"`
	Distance       float64          `json:"distance"`
	Duration       float64          `json:"duration"`
	Calories       int              `json:"calories"`
	AverageHR      int              `json:"averageHR"`
}

type activitySummary struct {
	ID             int64   `json:"id"`
	Date           string  `json:"date"`
	Type           string  `json:"type"`
	Name           string  `json:"name"`
	DistanceMeters float64 `json:"distance_meters"`
	DurationSeconds float64 `json:"duration_seconds"`
	Calories       int     `json:"calories,omitempty"`
	AvgHR          int     `json:"avg_hr,omitempty"`
}

func (a activityListItem) startDate() string {
	if len(a.StartTimeLocal) >= 10 {
		return a.StartTimeLocal[:10]
	}
	return ""
}

func (a activityListItem) toSummary() activitySummary {
	return activitySummary{
		ID:              a.ActivityID,
		Date:            a.startDate(),
		Type:            a.ActivityType.TypeKey,
		Name:            a.ActivityName,
		DistanceMeters:  a.Distance,
		DurationSeconds: a.Duration,
		Calories:        a.Calories,
		AvgHR:           a.AverageHR,
	}
}

func passesActivityFilters(a activitySummary, after, before, activityType string) bool {
	if after != "" && a.Date != "" && a.Date < after {
		return false
	}
	if before != "" && a.Date != "" && a.Date > before {
		return false
	}
	if strings.TrimSpace(activityType) != "" {
		t := strings.ToLower(strings.TrimSpace(activityType))
		typ := strings.ToLower(a.Type)
		if typ != t && !strings.Contains(typ, t) {
			return false
		}
	}
	return true
}

type activityDetailSummary struct {
	ID             int64
	Name           string
	Type           string
	StartTimeLocal string
	DistanceMeters float64
	DurationSeconds float64
	Calories       int
	AvgHR          int
	MaxHR          int
	ElevationGain  float64
	VO2Max         float64
	TrainingLoad   float64
}

func summarizeActivityDetail(id int64, raw map[string]any) activityDetailSummary {
	s := activityDetailSummary{ID: id}
	s.Name, _ = raw["activityName"].(string)
	s.StartTimeLocal, _ = raw["startTimeLocal"].(string)

	if at, ok := raw["activityType"].(map[string]any); ok {
		if tk, ok := at["typeKey"].(string); ok {
			s.Type = tk
		}
	}
	s.DistanceMeters = floatFromAny(raw["distance"])
	s.DurationSeconds = floatFromAny(raw["duration"])
	s.Calories = intFromAny(raw["calories"])
	s.AvgHR = intFromAny(raw["averageHR"])
	s.MaxHR = intFromAny(raw["maxHR"])
	s.ElevationGain = floatFromAny(raw["elevationGain"])
	s.VO2Max = floatFromAny(raw["vO2MaxValue"])
	s.TrainingLoad = floatFromAny(raw["activityTrainingLoad"])

	return s
}

func floatFromAny(v any) float64 {
	switch t := v.(type) {
	case float64:
		return t
	case int:
		return float64(t)
	case int64:
		return float64(t)
	default:
		return 0
	}
}

func intFromAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case int64:
		return int(t)
	default:
		return 0
	}
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

