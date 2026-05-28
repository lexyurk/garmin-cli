package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	garminactivities "github.com/lexyurk/garmin-cli/internal/activities"
	garminhealth "github.com/lexyurk/garmin-cli/internal/health"
	"github.com/lexyurk/garmin-cli/internal/output"
	garmintraining "github.com/lexyurk/garmin-cli/internal/training"
	"github.com/spf13/cobra"
)

type todayJSON struct {
	Date           string  `json:"date"`
	Steps          *int    `json:"steps,omitempty"`
	RestingHR      *int    `json:"resting_hr,omitempty"`
	StressAvg      *int    `json:"stress_avg,omitempty"`
	BodyBattery    *int    `json:"body_battery,omitempty"`
	TrainingStatus string  `json:"training_status,omitempty"`
	Readiness      *int    `json:"readiness,omitempty"`
	ReadinessLevel string  `json:"readiness_level,omitempty"`
	LastActivity   string  `json:"last_activity,omitempty"`
	LastActivityKM float64 `json:"last_activity_km,omitempty"`
	LastActivityAt string  `json:"last_activity_at,omitempty"`
}

func NewTodayCmd(opts *globalOptions) *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "today",
		Short: "Today's snapshot: health, training status/readiness, latest activity",
		RunE: func(cmd *cobra.Command, args []string) error {
			d := strings.TrimSpace(date)
			if d == "" {
				d = time.Now().In(time.Local).Format("2006-01-02")
			} else if _, err := time.ParseInLocation("2006-01-02", d, time.Local); err != nil {
				return fmt.Errorf("invalid --date %q (expected YYYY-MM-DD)", date)
			}

			c, err := newAuthedClient(cmd, opts)
			if err != nil {
				return err
			}

			ctx := cmd.Context()

			var (
				summary    garminhealth.DailySummary
				summaryErr error
				status     garmintraining.StatusSummary
				readiness  garmintraining.ReadinessSummary
				latest     garminactivities.Summary
				wg         sync.WaitGroup
			)
			run := func(f func()) {
				wg.Add(1)
				go func() { defer wg.Done(); f() }()
			}
			// The daily summary is the anchor: its error drives auth handling.
			run(func() { summary, summaryErr = garminhealth.GetDailySummary(ctx, c, d) })
			// The rest are best-effort — a missing section just renders as "—".
			run(func() { status, _ = garmintraining.GetStatus(ctx, c, d) })
			run(func() { readiness, _ = garmintraining.GetReadiness(ctx, c, d) })
			run(func() { latest, _ = garminactivities.Latest(ctx, c) })
			wg.Wait()

			if summaryErr != nil {
				return handleAuthedErrorTo(cmd.ErrOrStderr(), opts, summaryErr)
			}

			if opts.Format == "json" {
				return output.JSONTo(cmd.OutOrStdout(), todayJSON{
					Date:           summary.CalendarDateOr(d),
					Steps:          summary.TotalSteps,
					RestingHR:      summary.RestingHeartRate,
					StressAvg:      summary.AverageStressLevel,
					BodyBattery:    summary.BodyBatteryMostRecentValue,
					TrainingStatus: status.StatusPhrase,
					Readiness:      readiness.Score,
					ReadinessLevel: readiness.Level,
					LastActivity:   latest.Name,
					LastActivityKM: latest.DistanceMeters / 1000.0,
					LastActivityAt: latest.StartTimeLocal,
				})
			}

			lastActivity := "—"
			if strings.TrimSpace(latest.Name) != "" || latest.ID != 0 {
				lastActivity = fmt.Sprintf("%s (%s, %s km)", orDash(latest.Name), orDash(latest.Type), formatDistanceKM(latest.DistanceMeters))
			}
			readinessStr := formatMaybeInt(readiness.Score)
			if strings.TrimSpace(readiness.Level) != "" {
				readinessStr = strings.TrimSpace(readinessStr + " " + readiness.Level)
			}

			return renderKVTo(cmd.OutOrStdout(), opts.Format, "Today ("+summary.CalendarDateOr(d)+")", map[string]string{
				"steps":            formatMaybeInt(summary.TotalSteps),
				"resting_hr":       formatMaybeInt(summary.RestingHeartRate),
				"stress_avg":       formatMaybeInt(summary.AverageStressLevel),
				"body_battery":     formatMaybeInt(summary.BodyBatteryMostRecentValue),
				"training_status":  orDash(status.StatusPhrase),
				"readiness":        orDash(readinessStr),
				"last_activity":    lastActivity,
				"last_activity_at": orDash(latest.StartTimeLocal),
			})
		},
	}

	cmd.Flags().StringVar(&date, "date", "", "Date (YYYY-MM-DD, default: today)")
	return cmd
}
