package main

import (
	"fmt"
	"strconv"

	"github.com/lexyurk/garmin-cli/internal/output"
	"github.com/lexyurk/garmin-cli/internal/workouts"
	"github.com/spf13/cobra"
)

func NewWorkoutsCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workouts",
		Short: "Plan and manage structured workouts",
	}

	cmd.AddCommand(
		newWorkoutsListCmd(opts),
		newWorkoutsGetCmd(opts),
	)

	return cmd
}

func newWorkoutsListCmd(opts *globalOptions) *cobra.Command {
	var limit int

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List saved workouts",
		RunE: func(cmd *cobra.Command, args []string) error {
			if limit <= 0 {
				return fmt.Errorf("--limit must be > 0")
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			out, err := workouts.List(ctx, c, limit)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), out)
			}

			rows := make([][]string, 0, len(out))
			for _, w := range out {
				rows = append(rows, []string{
					strconv.FormatInt(w.WorkoutID, 10),
					w.Name,
					orDash(w.Sport),
					formatDurationSecondsFloat(w.DurationSecs),
					formatDistanceKM(w.DistanceM),
				})
			}
			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"id", "name", "sport", "duration", "dist_km"}, rows)
		},
	}

	cmd.Flags().IntVar(&limit, "limit", 20, "Number of workouts to return")
	return cmd
}

func newWorkoutsGetCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [workout-id]",
		Short: "Get workout details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid workout id %q", args[0])
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			raw, err := workouts.GetRaw(ctx, c, id)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), raw)
			}

			s := workouts.SummarizeRaw(id, raw)
			return renderKVTo(cmd.OutOrStdout(), opts.Format, "Workout", map[string]string{
				"id":          strconv.FormatInt(s.WorkoutID, 10),
				"name":        orDash(s.Name),
				"sport":       orDash(s.Sport),
				"duration":    formatDurationSecondsFloat(s.DurationSecs),
				"distance_km": formatDistanceKM(s.DistanceM),
				"steps":       strconv.Itoa(workouts.CountSteps(raw)),
				"description": orDash(s.Description),
			})
		},
	}
	return cmd
}
