package health

import (
	"context"
	"net/url"

	"github.com/lexyurk/garmin-cli/internal/client"
)

type DailySummary struct {
	CalendarDate string `json:"calendarDate"`

	TotalSteps          *int `json:"totalSteps"`
	DailyStepGoal       *int `json:"dailyStepGoal"`
	TotalDistanceMeters *int `json:"totalDistanceMeters"`

	MinHeartRate     *int `json:"minHeartRate"`
	MaxHeartRate     *int `json:"maxHeartRate"`
	RestingHeartRate *int `json:"restingHeartRate"`

	AverageStressLevel *int   `json:"averageStressLevel"`
	MaxStressLevel     *int   `json:"maxStressLevel"`
	StressQualifier    string `json:"stressQualifier"`

	BodyBatteryChargedValue    *int `json:"bodyBatteryChargedValue"`
	BodyBatteryDrainedValue    *int `json:"bodyBatteryDrainedValue"`
	BodyBatteryHighestValue    *int `json:"bodyBatteryHighestValue"`
	BodyBatteryLowestValue     *int `json:"bodyBatteryLowestValue"`
	BodyBatteryMostRecentValue *int `json:"bodyBatteryMostRecentValue"`
}

func (d DailySummary) CalendarDateOr(fallback string) string {
	if d.CalendarDate != "" {
		return d.CalendarDate
	}
	return fallback
}

func GetDailySummary(ctx context.Context, c *client.Client, date string) (DailySummary, error) {
	var resp DailySummary
	q := url.Values{"calendarDate": {date}}
	err := c.GetJSON(ctx, "/usersummary-service/usersummary/daily/", q, &resp)
	return resp, err
}

