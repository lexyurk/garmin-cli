package main

import (
	"strconv"

	"github.com/lexyurk/garmin-cli/internal/output"
	"github.com/lexyurk/garmin-cli/internal/profile"
	"github.com/spf13/cobra"
)

func NewProfileCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Show your Garmin Connect profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			p, err := profile.Get(ctx, c)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), p)
			}

			id := "—"
			if p.ProfileID != 0 {
				id = strconv.FormatInt(p.ProfileID, 10)
			}
			return renderKVTo(cmd.OutOrStdout(), opts.Format, "Profile", map[string]string{
				"profile_id":   id,
				"display_name": orDash(p.DisplayName),
				"full_name":    orDash(p.FullName),
				"user_name":    orDash(p.UserName),
				"location":     orDash(p.Location),
			})
		},
	}
	return cmd
}
