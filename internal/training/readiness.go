package training

import (
	"context"
	"fmt"

	"github.com/lexyurk/garmin-cli/internal/client"
)

type ReadinessEntry struct {
	CalendarDate     string `json:"calendarDate"`
	Timestamp        string `json:"timestamp"`
	Level            string `json:"level"`
	Score            *int   `json:"score"`
	SleepScore       *int   `json:"sleepScore"`
	HRVWeeklyAverage *int   `json:"hrvWeeklyAverage"`
	AcuteLoad        *int   `json:"acuteLoad"`
	RecoveryTime     *int   `json:"recoveryTime"`
}

type ReadinessSummary struct {
	Date             string `json:"date"`
	Level            string `json:"level,omitempty"`
	Score            *int   `json:"score,omitempty"`
	SleepScore       *int   `json:"sleep_score,omitempty"`
	HRVWeeklyAverage *int   `json:"hrv_weekly_avg,omitempty"`
	AcuteLoad        *int   `json:"acute_load,omitempty"`
	RecoveryTime     *int   `json:"recovery_time_seconds,omitempty"`
}

func GetReadiness(ctx context.Context, c *client.Client, date string) (ReadinessSummary, error) {
	var entries []ReadinessEntry
	if err := c.GetJSON(ctx, fmt.Sprintf("/metrics-service/metrics/trainingreadiness/%s", date), nil, &entries); err != nil {
		return ReadinessSummary{}, err
	}
	return summarizeReadiness(date, entries), nil
}

func summarizeReadiness(fallbackDate string, entries []ReadinessEntry) ReadinessSummary {
	best := ReadinessEntry{}
	for _, e := range entries {
		if e.Timestamp > best.Timestamp {
			best = e
		}
	}
	date := best.CalendarDate
	if date == "" {
		date = fallbackDate
	}
	return ReadinessSummary{
		Date:             date,
		Level:            best.Level,
		Score:            best.Score,
		SleepScore:       best.SleepScore,
		HRVWeeklyAverage: best.HRVWeeklyAverage,
		AcuteLoad:        best.AcuteLoad,
		RecoveryTime:     best.RecoveryTime,
	}
}

