package training

import (
	"context"
	"fmt"

	"github.com/lexyurk/garmin-cli/internal/client"
)

type HRVResponse struct {
	HRVSummary HRVSummaryDTO `json:"hrvSummary"`
}

type HRVSummaryDTO struct {
	CalendarDate string         `json:"calendarDate"`
	WeeklyAvg    *int           `json:"weeklyAvg"`
	LastNightAvg *int           `json:"lastNightAvg"`
	Status       string         `json:"status"`
	Baseline     map[string]any `json:"baseline"`
}

type HRVSummary struct {
	Date                  string   `json:"date"`
	WeeklyAvg             *int     `json:"weekly_avg,omitempty"`
	LastNightAvg          *int     `json:"last_night_avg,omitempty"`
	Status                string   `json:"status,omitempty"`
	BaselineLowUpper      *float64 `json:"baseline_low_upper,omitempty"`
	BaselineBalancedUpper *float64 `json:"baseline_balanced_upper,omitempty"`
}

func GetHRV(ctx context.Context, c *client.Client, date string) (HRVSummary, error) {
	var resp HRVResponse
	if err := c.GetJSON(ctx, fmt.Sprintf("/hrv-service/hrv/%s", date), nil, &resp); err != nil {
		return HRVSummary{}, err
	}
	return resp.ToSummary(date), nil
}

func (r HRVResponse) ToSummary(fallbackDate string) HRVSummary {
	date := r.HRVSummary.CalendarDate
	if date == "" {
		date = fallbackDate
	}
	s := HRVSummary{
		Date:         date,
		WeeklyAvg:    r.HRVSummary.WeeklyAvg,
		LastNightAvg: r.HRVSummary.LastNightAvg,
		Status:       r.HRVSummary.Status,
	}
	if v, ok := r.HRVSummary.Baseline["lowUpper"].(float64); ok {
		s.BaselineLowUpper = &v
	}
	if v, ok := r.HRVSummary.Baseline["balancedUpper"].(float64); ok {
		s.BaselineBalancedUpper = &v
	}
	return s
}
