package main

import (
	"strconv"

	"github.com/lexyurk/garmin-cli/internal/devices"
	"github.com/lexyurk/garmin-cli/internal/output"
	"github.com/spf13/cobra"
)

func NewDevicesCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "devices",
		Short: "Registered Garmin devices",
	}

	cmd.AddCommand(newDevicesListCmd(opts))
	return cmd
}

func newDevicesListCmd(opts *globalOptions) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List registered devices",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()
			if opts.Format == "json" {
				raw, err := devices.ListRaw(ctx, c)
				if err != nil {
					return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
				}
				return output.JSONTo(cmd.OutOrStdout(), raw)
			}

			list, err := devices.List(ctx, c)
			if err != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, err)
			}
			rows := make([][]string, 0, len(list))
			for _, d := range list {
				id := "—"
				if d.DeviceID != 0 {
					id = strconv.FormatInt(d.DeviceID, 10)
				}
				rows = append(rows, []string{id, orDash(d.Name), orDash(d.SerialNumber), orDash(d.PartNumber)})
			}
			return renderTableTo(cmd.OutOrStdout(), opts.Format, []string{"device_id", "name", "serial", "part_no"}, rows)
		},
	}
	return cmd
}
