package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/lexyurk/garmin-cli/internal/calendar"
	"github.com/lexyurk/garmin-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewCalendarCmd(opts *globalOptions) *cobra.Command {
	var month string
	var itemType string

	cmd := &cobra.Command{
		Use:   "calendar",
		Short: "View the training calendar (scheduled workouts and activities)",
		RunE: func(cmd *cobra.Command, args []string) error {
			year, mon, err := resolveCalendarMonth(month, time.Now())
			if err != nil {
				return err
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			items, err := calendar.Month(ctx, c, year, mon)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}
			items = calendar.FilterByType(items, itemType)

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), items)
			}
			rows := make([][]string, 0, len(items))
			for _, it := range items {
				ref := "—"
				switch {
				case it.WorkoutID != 0:
					ref = "workout:" + strconv.FormatInt(it.WorkoutID, 10)
				case it.ActivityID != 0:
					ref = "activity:" + strconv.FormatInt(it.ActivityID, 10)
				}
				rows = append(rows, []string{orDash(it.Date), orDash(it.Type), orDash(it.Title), ref})
			}
			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"date", "type", "title", "ref"}, rows)
		},
	}

	cmd.Flags().StringVar(&month, "month", "", "Month to view (YYYY-MM, default: current month)")
	cmd.Flags().StringVar(&itemType, "type", "", "Filter by item type (e.g. workout, activity)")
	return cmd
}

func resolveCalendarMonth(month string, now time.Time) (int, int, error) {
	month = strings.TrimSpace(month)
	if month == "" {
		n := now.In(time.Local)
		return n.Year(), int(n.Month()), nil
	}
	t, err := time.Parse("2006-01", month)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid --month %q (expected YYYY-MM)", month)
	}
	return t.Year(), int(t.Month()), nil
}
