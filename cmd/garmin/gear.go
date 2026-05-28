package main

import (
	"fmt"
	"strconv"

	"github.com/lexyurk/garmin-cli/internal/gear"
	"github.com/lexyurk/garmin-cli/internal/output"
	"github.com/lexyurk/garmin-cli/internal/profile"
	"github.com/spf13/cobra"
)

func NewGearCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gear",
		Short: "Manage gear (shoes, bikes)",
	}

	cmd.AddCommand(
		newGearListCmd(opts),
		newGearGetCmd(opts),
		newGearStatsCmd(opts),
		newGearAddCmd(opts),
		newGearRetireCmd(opts),
		newGearRestoreCmd(opts),
		newGearLinkCmd(opts),
		newGearUnlinkCmd(opts),
		newGearForActivityCmd(opts),
	)

	return cmd
}

func newGearLinkCmd(opts *globalOptions) *cobra.Command {
	return newGearLinkLikeCmd(opts, "link", "Assign a gear item to an activity", false)
}

func newGearUnlinkCmd(opts *globalOptions) *cobra.Command {
	return newGearLinkLikeCmd(opts, "unlink", "Remove a gear item from an activity", true)
}

func newGearLinkLikeCmd(opts *globalOptions, use, short string, unlink bool) *cobra.Command {
	return &cobra.Command{
		Use:   use + " [uuid] [activity-id]",
		Short: short,
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			activityID, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid activity id %q", args[1])
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			if unlink {
				err = gear.Unlink(ctx, c, args[0], activityID)
			} else {
				err = gear.Link(ctx, c, args[0], activityID)
			}
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			action := "linked"
			if unlink {
				action = "unlinked"
			}
			return renderKVTo(cmd.OutOrStdout(), opts.Format, "Gear "+action, map[string]string{
				"gear":        args[0],
				"activity_id": args[1],
				"status":      action,
			})
		},
	}
}

func newGearForActivityCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "for-activity [activity-id]",
		Short: "Show gear linked to an activity",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			activityID, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid activity id %q", args[0])
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			gears, err := gear.ForActivity(ctx, c, activityID)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), gears)
			}
			rows := make([][]string, 0, len(gears))
			for _, g := range gears {
				rows = append(rows, []string{g.UUID, g.Name, orDash(g.Type), orDash(g.Status)})
			}
			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"uuid", "name", "type", "status"}, rows)
		},
	}
	return cmd
}

func newGearAddCmd(opts *globalOptions) *cobra.Command {
	var gearType string
	var name string
	var make_ string
	var model string
	var maxKM float64
	var begin string

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new gear item (e.g. a pair of shoes)",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			pk, err := profile.UserProfilePK(ctx, c)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			g, err := gear.Create(ctx, c, pk, gear.CreateOptions{
				Type:      gearType,
				Name:      name,
				Make:      make_,
				Model:     model,
				MaxMeters: maxKM * 1000,
				DateBegin: begin,
			})
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), g)
			}
			return renderKVTo(cmd.OutOrStdout(), opts.Format, "Gear added", map[string]string{
				"uuid":   orDash(g.UUID),
				"name":   orDash(g.Name),
				"type":   orDash(g.Type),
				"status": orDash(g.Status),
				"max_km": formatDistanceKM(g.MaxMeters),
			})
		},
	}
	cmd.Flags().StringVar(&gearType, "type", "Shoes", "Gear type (Shoes, Bike, Other)")
	cmd.Flags().StringVar(&name, "name", "", "Display name (required)")
	cmd.Flags().StringVar(&make_, "make", "", "Manufacturer")
	cmd.Flags().StringVar(&model, "model", "", "Model")
	cmd.Flags().Float64Var(&maxKM, "max-km", 0, "Retirement distance in km (0 = none)")
	cmd.Flags().StringVar(&begin, "begin", "", "Start date (YYYY-MM-DD, default: today)")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func newGearRetireCmd(opts *globalOptions) *cobra.Command {
	return newGearStatusCmd(opts, "retire", "Retire a gear item", "retired", "Gear retired")
}

func newGearRestoreCmd(opts *globalOptions) *cobra.Command {
	return newGearStatusCmd(opts, "restore", "Restore (un-retire) a gear item", "active", "Gear restored")
}

func newGearStatusCmd(opts *globalOptions, use, short, status, title string) *cobra.Command {
	return &cobra.Command{
		Use:   use + " [uuid]",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			pk, err := profile.UserProfilePK(ctx, c)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			g, err := gear.SetStatus(ctx, c, pk, args[0], status)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), g)
			}
			return renderKVTo(cmd.OutOrStdout(), opts.Format, title, map[string]string{
				"uuid":   orDash(g.UUID),
				"name":   orDash(g.Name),
				"status": orDash(g.Status),
			})
		},
	}
}

func newGearListCmd(opts *globalOptions) *cobra.Command {
	var showAll bool
	var retired bool
	var withStats bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List gear (active by default)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if showAll && retired {
				return fmt.Errorf("use either --all or --retired (not both)")
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			pk, err := profile.UserProfilePK(ctx, c)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			gears, err := gear.List(ctx, c, pk)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			status := "active"
			switch {
			case showAll:
				status = "all"
			case retired:
				status = "retired"
			}
			gears = gear.FilterByStatus(gears, status)

			if withStats {
				gears = gear.WithStats(ctx, c, gears)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), gears)
			}

			headers := []string{"uuid", "name", "type", "status", "max_km"}
			if withStats {
				headers = append(headers, "total_km", "acts")
			}
			rows := make([][]string, 0, len(gears))
			for _, g := range gears {
				row := []string{g.UUID, g.Name, orDash(g.Type), orDash(g.Status), formatDistanceKM(g.MaxMeters)}
				if withStats {
					row = append(row, formatMetersPtrKM(g.TotalMeters), formatMaybeInt(g.Activities))
				}
				rows = append(rows, row)
			}
			return renderTableTo(cmd.OutOrStdout(), opts.Format, headers, rows)
		},
	}

	cmd.Flags().BoolVar(&showAll, "all", false, "Include retired gear")
	cmd.Flags().BoolVar(&retired, "retired", false, "Show only retired gear")
	cmd.Flags().BoolVar(&withStats, "stats", false, "Fetch cumulative distance/activities per gear (slower)")
	return cmd
}

func newGearGetCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [uuid]",
		Short: "Get gear details (with cumulative stats)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			pk, err := profile.UserProfilePK(ctx, c)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			g, err := gear.Get(ctx, c, pk, args[0])
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), g)
			}
			return renderKVTo(cmd.OutOrStdout(), opts.Format, "Gear", map[string]string{
				"uuid":             g.UUID,
				"name":             orDash(g.Name),
				"make":             orDash(g.Make),
				"model":            orDash(g.Model),
				"type":             orDash(g.Type),
				"status":           orDash(g.Status),
				"max_km":           formatDistanceKM(g.MaxMeters),
				"total_km":         formatMetersPtrKM(g.TotalMeters),
				"total_activities": formatMaybeInt(g.Activities),
				"date_begin":       orDash(g.DateBegin),
				"date_end":         orDash(g.DateEnd),
			})
		},
	}
	return cmd
}

func newGearStatsCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stats [uuid]",
		Short: "Cumulative distance/activities for a gear item",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			st, err := gear.GetStats(ctx, c, args[0])
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), st)
			}
			return renderKVTo(cmd.OutOrStdout(), opts.Format, "Gear stats", map[string]string{
				"uuid":             args[0],
				"total_km":         formatDistanceKM(st.TotalMeters),
				"total_activities": fmt.Sprintf("%d", st.TotalActivities),
			})
		},
	}
	return cmd
}

func formatMetersPtrKM(m *float64) string {
	if m == nil {
		return "—"
	}
	return formatDistanceKM(*m)
}
