package main

import (
	"strconv"

	"github.com/lexyurk/garmin-cli/internal/output"
	"github.com/lexyurk/garmin-cli/internal/profile"
	"github.com/lexyurk/garmin-cli/internal/records"
	"github.com/spf13/cobra"
)

func NewRecordsCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "records",
		Short: "Personal records (PRs)",
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
				raw, err := records.ListRaw(ctx, c, p.DisplayName)
				if err != nil {
					return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
				}
				return output.JSONTo(cmd.OutOrStdout(), raw)
			}

			list, err := records.List(ctx, c, p.DisplayName)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}
			rows := make([][]string, 0, len(list))
			for _, r := range list {
				actID := "—"
				if r.ActivityID != 0 {
					actID = strconv.FormatInt(r.ActivityID, 10)
				}
				rows = append(rows, []string{
					strconv.Itoa(r.TypeID),
					r.Label,
					strconv.FormatFloat(r.Value, 'f', -1, 64),
					actID,
					orDash(r.ActivityName),
				})
			}
			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"type_id", "label", "value", "activity_id", "activity"}, rows)
		},
	}
	return cmd
}
