package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

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
		newWorkoutsCreateCmd(opts),
		newWorkoutsUpdateCmd(opts),
		newWorkoutsDeleteCmd(opts),
		newWorkoutsScheduleCmd(opts),
	)

	return cmd
}

func newWorkoutsCreateCmd(opts *globalOptions) *cobra.Command {
	var name string
	var sport string
	var description string
	var steps []string
	var fromJSON string
	var schedule string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a structured workout",
		Long: `Create a structured workout from step specifications.

Each --step is either an executable step or a repeat block:

  --step "warmup 10min"
  --step "interval 800m"
  --step "recovery 90s"
  --step "interval lap"
  --step "4x(interval 800m; recovery 2min)"

Durations use min/s/sec (time) or m/km (distance), or the word "lap".
Step types: warmup, cooldown, interval, recovery, rest, other.

Example:
  garmin workouts create --name "4x800m" \
    --step "warmup 10min" \
    --step "4x(interval 800m; recovery 2min)" \
    --step "cooldown 5min"

Advanced users can pass a full workout JSON body with --from-json instead.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			payload, err := buildWorkoutPayload(name, description, sport, steps, fromJSON)
			if err != nil {
				return err
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			s, err := workouts.Create(ctx, c, payload)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			scheduled := ""
			if strings.TrimSpace(schedule) != "" {
				if _, err := workouts.Schedule(ctx, c, s.WorkoutID, strings.TrimSpace(schedule)); err != nil {
					return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
				}
				scheduled = strings.TrimSpace(schedule)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), s)
			}
			fields := map[string]string{
				"id":     strconv.FormatInt(s.WorkoutID, 10),
				"name":   orDash(s.Name),
				"sport":  orDash(s.Sport),
				"status": "created",
			}
			if scheduled != "" {
				fields["scheduled"] = scheduled
			}
			return renderKVTo(cmd.OutOrStdout(), opts.Format, "Workout created", fields)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Workout name (required unless --from-json)")
	cmd.Flags().StringVar(&sport, "sport", "running", "Sport: running, cycling")
	cmd.Flags().StringVar(&description, "description", "", "Optional description")
	cmd.Flags().StringArrayVar(&steps, "step", nil, "Workout step (repeatable); see --help")
	cmd.Flags().StringVar(&fromJSON, "from-json", "", "Path to a full workout JSON body (overrides --step)")
	cmd.Flags().StringVar(&schedule, "schedule", "", "Also schedule onto this date (YYYY-MM-DD)")
	return cmd
}

func buildWorkoutPayload(name, description, sport string, steps []string, fromJSON string) (map[string]any, error) {
	if strings.TrimSpace(fromJSON) != "" {
		data, err := os.ReadFile(fromJSON)
		if err != nil {
			return nil, err
		}
		var payload map[string]any
		if err := json.Unmarshal(data, &payload); err != nil {
			return nil, fmt.Errorf("parsing %s: %w", fromJSON, err)
		}
		return payload, nil
	}

	if len(steps) == 0 {
		return nil, fmt.Errorf("provide at least one --step (or use --from-json)")
	}
	parsed, err := workouts.ParseSteps(steps)
	if err != nil {
		return nil, err
	}
	return workouts.BuildPayload(name, description, sport, parsed)
}

func newWorkoutsScheduleCmd(opts *globalOptions) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "schedule [workout-id]",
		Short: "Schedule a workout onto a date in your training calendar",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid workout id %q", args[0])
			}
			if strings.TrimSpace(date) == "" {
				return fmt.Errorf("--date is required (YYYY-MM-DD)")
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			res, err := workouts.Schedule(ctx, c, id, strings.TrimSpace(date))
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), res)
			}
			fields := map[string]string{
				"workout_id": strconv.FormatInt(res.WorkoutID, 10),
				"date":       res.Date,
				"status":     "scheduled",
			}
			if res.WorkoutScheduleID != 0 {
				fields["schedule_id"] = strconv.FormatInt(res.WorkoutScheduleID, 10)
			}
			return renderKVTo(cmd.OutOrStdout(), opts.Format, "Workout scheduled", fields)
		},
	}

	cmd.Flags().StringVar(&date, "date", "", "Date to schedule (YYYY-MM-DD)")
	return cmd
}

func newWorkoutsUpdateCmd(opts *globalOptions) *cobra.Command {
	var name string
	var description string

	cmd := &cobra.Command{
		Use:   "update [workout-id]",
		Short: "Update a workout's name or description",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid workout id %q", args[0])
			}

			nameSet := cmd.Flags().Changed("name")
			descSet := cmd.Flags().Changed("description")
			if !nameSet && !descSet {
				return fmt.Errorf("specify at least one of --name, --description")
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			up := workouts.UpdateOptions{}
			if nameSet {
				up.Name = &name
			}
			if descSet {
				up.Description = &description
			}

			ctx := cmd.Context()
			s, err := workouts.Update(ctx, c, id, up)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), s)
			}
			return renderKVTo(cmd.OutOrStdout(), opts.Format, "Workout updated", map[string]string{
				"id":          strconv.FormatInt(s.WorkoutID, 10),
				"name":        orDash(s.Name),
				"description": orDash(s.Description),
			})
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "New workout name")
	cmd.Flags().StringVar(&description, "description", "", "New description")
	return cmd
}

func newWorkoutsDeleteCmd(opts *globalOptions) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete [workout-id]",
		Short: "Delete a workout (irreversible)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid workout id %q", args[0])
			}

			if !confirmDestructive(cmd, fmt.Sprintf("Delete workout %s? This cannot be undone.", args[0]), force) {
				return fmt.Errorf("aborted: pass --force to delete non-interactively")
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			if err := workouts.Delete(ctx, c, id); err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			return renderKVTo(cmd.OutOrStdout(), opts.Format, "Workout deleted", map[string]string{
				"id":     args[0],
				"status": "deleted",
			})
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Skip confirmation prompt")
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
