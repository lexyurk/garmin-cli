package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewTrainingCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "training",
		Short: "Training metrics (status, VO2max, HRV, readiness)",
	}

	cmd.AddCommand(
		newTrainingStatusCmd(opts),
		newTrainingReadinessCmd(opts),
		newTrainingVo2maxCmd(opts),
		newTrainingHrvCmd(opts),
	)

	return cmd
}

func newTrainingStatusCmd(opts *globalOptions) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Training status (Productive, Peaking, etc.)",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = opts
			_ = date
			fmt.Println("TODO: garmin training status")
			return nil
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	return cmd
}

func newTrainingReadinessCmd(opts *globalOptions) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "readiness",
		Short: "Training readiness score (0-100)",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = opts
			_ = date
			fmt.Println("TODO: garmin training readiness")
			return nil
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	return cmd
}

func newTrainingVo2maxCmd(opts *globalOptions) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "vo2max",
		Short: "VO2 max estimates",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = opts
			_ = date
			fmt.Println("TODO: garmin training vo2max")
			return nil
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	return cmd
}

func newTrainingHrvCmd(opts *globalOptions) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "hrv",
		Short: "Heart rate variability",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = opts
			_ = date
			fmt.Println("TODO: garmin training hrv")
			return nil
		},
	}
	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	return cmd
}

