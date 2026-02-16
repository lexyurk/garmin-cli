package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func NewAuthCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authentication commands",
	}

	cmd.AddCommand(
		newAuthLoginCmd(opts),
		newAuthStatusCmd(opts),
		newAuthLogoutCmd(opts),
	)

	return cmd
}

func newAuthLoginCmd(opts *globalOptions) *cobra.Command {
	var email string
	var password string

	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to Garmin Connect",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = opts
			_ = email
			_ = password
			fmt.Println("TODO: garmin auth login")
			return nil
		},
	}

	cmd.Flags().StringVar(&email, "email", "", "Garmin Connect email")
	cmd.Flags().StringVar(&password, "password", "", "Garmin Connect password")

	return cmd
}

func newAuthStatusCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = opts
			fmt.Println("TODO: garmin auth status")
			return nil
		},
	}
	return cmd
}

func newAuthLogoutCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Clear stored tokens",
		RunE: func(cmd *cobra.Command, args []string) error {
			_ = opts
			fmt.Println("TODO: garmin auth logout")
			return nil
		},
	}
	return cmd
}

