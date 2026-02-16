package main

import (
	"fmt"

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
			_ = opts
			_ = limit
			_ = after
			_ = before
			_ = activityType
			fmt.Println("TODO: garmin activities list")
			return nil
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
			_ = opts
			_ = details
			fmt.Printf("TODO: garmin activities get %s\n", args[0])
			return nil
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

