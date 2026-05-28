package main

import (
	"fmt"

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
	)

	return cmd
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
