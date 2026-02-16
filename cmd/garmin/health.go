package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewHealthCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Health data (sleep, HR, steps, stress, body battery)",
	}

	cmd.AddCommand(
		newHealthSleepCmd(opts),
		newHealthHeartRateCmd(opts),
		newHealthStepsCmd(opts),
		newHealthStressCmd(opts),
		newHealthBodyBatteryCmd(opts),
	)

	return cmd
}

func newHealthSleepCmd(opts *globalOptions) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "sleep",
		Short: "Sleep data",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = opts
			_ = date
			fmt.Println("TODO: garmin health sleep")
			return nil
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	return cmd
}

func newHealthHeartRateCmd(opts *globalOptions) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "heart-rate",
		Short: "Heart rate data",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = opts
			_ = date
			fmt.Println("TODO: garmin health heart-rate")
			return nil
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	return cmd
}

func newHealthStepsCmd(opts *globalOptions) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "steps",
		Short: "Step count",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = opts
			_ = date
			fmt.Println("TODO: garmin health steps")
			return nil
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	return cmd
}

func newHealthStressCmd(opts *globalOptions) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "stress",
		Short: "Stress levels",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = opts
			_ = date
			fmt.Println("TODO: garmin health stress")
			return nil
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	return cmd
}

func newHealthBodyBatteryCmd(opts *globalOptions) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "body-battery",
		Short: "Body battery",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = opts
			_ = date
			fmt.Println("TODO: garmin health body-battery")
			return nil
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	return cmd
}

