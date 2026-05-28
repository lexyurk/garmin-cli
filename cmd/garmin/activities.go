package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	garminactivities "github.com/lexyurk/garmin-cli/internal/activities"
	"github.com/lexyurk/garmin-cli/internal/output"
	"github.com/spf13/cobra"
	"golang.org/x/term"
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
		newActivitiesExportCmd(opts),
		newActivitiesUpdateCmd(opts),
		newActivitiesDeleteCmd(opts),
	)

	return cmd
}

func newActivitiesUpdateCmd(opts *globalOptions) *cobra.Command {
	var name string
	var description string
	var activityType string

	cmd := &cobra.Command{
		Use:   "update [activity-id]",
		Short: "Update an activity's name, description, or type",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid activity id %q", args[0])
			}

			nameSet := cmd.Flags().Changed("name")
			descSet := cmd.Flags().Changed("description")
			typeSet := cmd.Flags().Changed("type")
			if !nameSet && !descSet && !typeSet {
				return fmt.Errorf("specify at least one of --name, --description, --type")
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			up := garminactivities.UpdateOptions{}
			if nameSet {
				up.Name = &name
			}
			if descSet {
				up.Description = &description
			}
			if typeSet {
				types, err := garminactivities.GetActivityTypes(ctx, c)
				if err != nil {
					return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
				}
				t, err := garminactivities.ResolveType(types, activityType)
				if err != nil {
					return err
				}
				up.Type = &t
			}

			if err := garminactivities.Update(ctx, c, id, up); err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			fields := map[string]string{"id": args[0], "status": "updated"}
			if nameSet {
				fields["name"] = name
			}
			if descSet {
				fields["description"] = description
			}
			if typeSet {
				fields["type"] = activityType
			}
			return renderKVTo(cmd.OutOrStdout(), opts.Format, "Activity updated", fields)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New activity name")
	cmd.Flags().StringVar(&description, "description", "", "New description")
	cmd.Flags().StringVar(&activityType, "type", "", "New activity type key (e.g. running, cycling)")
	return cmd
}

func newActivitiesDeleteCmd(opts *globalOptions) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete [activity-id]",
		Short: "Delete an activity (irreversible)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid activity id %q", args[0])
			}

			if !confirmDestructive(cmd, fmt.Sprintf("Delete activity %s? This cannot be undone.", args[0]), force) {
				return fmt.Errorf("aborted: pass --force to delete non-interactively")
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			if err := garminactivities.Delete(ctx, c, id); err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			return renderKVTo(cmd.OutOrStdout(), opts.Format, "Activity deleted", map[string]string{
				"id":     args[0],
				"status": "deleted",
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
	return cmd
}

func newActivitiesListCmd(opts *globalOptions) *cobra.Command {
	var limit int
	var date string
	var after string
	var before string
	var from string
	var to string
	var activityType string
	var days int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List activities",
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("--limit must be > 0")
			}

			afterResolved, beforeResolved, err := resolveActivitiesDateFilters(date, after, before, from, to, days, time.Now())
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
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&after, "after", "", "Activities on/after date (YYYY-MM-DD, inclusive)")
	_ = cmd.Flags().MarkDeprecated("after", "use --from instead")
	cmd.Flags().StringVar(&before, "before", "", "Activities on/before date (YYYY-MM-DD, inclusive)")
	_ = cmd.Flags().MarkDeprecated("before", "use --to instead")
	cmd.Flags().StringVar(&from, "from", "", "Start date (YYYY-MM-DD, inclusive)")
	cmd.Flags().StringVar(&to, "to", "", "End date (YYYY-MM-DD, inclusive)")
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

func newActivitiesExportCmd(opts *globalOptions) *cobra.Command {
	var exportType string
	var outPath string
	var force bool

	cmd := &cobra.Command{
		Use:   "export [activity-id]",
		Short: "Download an activity file (GPX/TCX/original)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid activity id %q", args[0])
			}

			t := strings.ToLower(strings.TrimSpace(exportType))
			switch t {
			case "", "gpx":
				t = "gpx"
			case "tcx":
				// ok
			case "fit":
				// Garmin calls this "original".
				t = "original"
			case "original":
				// ok
			default:
				return fmt.Errorf("unsupported --type %q (supported: gpx, tcx, fit, original)", exportType)
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			w := cmd.OutOrStdout()
			var f *os.File
			if strings.TrimSpace(outPath) != "" {
				if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
					return err
				}
				flags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
				if !force {
					flags = os.O_CREATE | os.O_WRONLY | os.O_EXCL
				}
				f, err = os.OpenFile(outPath, flags, 0o644)
				if err != nil {
					return err
				}
				defer f.Close()
				w = f
			} else if outf, ok := w.(*os.File); ok && term.IsTerminal(int(outf.Fd())) {
				return fmt.Errorf("refusing to write activity file to terminal; use --out or redirect stdout")
			}

			ctx := cmd.Context()
			if err := garminactivities.Export(ctx, c, id, garminactivities.ExportType(t), w); err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			// If writing to a file, print a small confirmation to stderr (stdout stays clean).
			if f != nil && !opts.Quiet {
				_, _ = io.WriteString(cmd.ErrOrStderr(), "downloaded\n")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&exportType, "type", "gpx", "Export type: gpx, tcx, fit, original")
	cmd.Flags().StringVar(&outPath, "out", "", "Write to file instead of stdout")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite output file if it exists")
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

func resolveActivitiesDateFilters(date, after, before, from, to string, days int, now time.Time) (string, string, error) {
	if days < 0 {
		return "", "", fmt.Errorf("--days must be >= 0")
	}

	date = strings.TrimSpace(date)
	after = strings.TrimSpace(after)
	before = strings.TrimSpace(before)
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)

	if date != "" && (after != "" || before != "" || from != "" || to != "" || days > 0) {
		return "", "", fmt.Errorf("use either --date or --after/--before/--from/--to (not both)")
	}
	if after != "" && from != "" {
		return "", "", fmt.Errorf("use either --after or --from (not both)")
	}
	if before != "" && to != "" {
		return "", "", fmt.Errorf("use either --before or --to (not both)")
	}

	// Normalize to "after/before" naming for the internal activities API.
	if after == "" {
		after = from
	}
	if before == "" {
		before = to
	}

	if days > 0 && (after != "" || before != "") {
		return "", "", fmt.Errorf("use either --days or --after/--before/--from/--to (not both)")
	}
	if date != "" {
		return date, date, nil
	}
	if days == 0 {
		return after, before, nil
	}

	end := now.In(time.Local).Format("2006-01-02")
	start := now.In(time.Local).AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	return start, end, nil
}
