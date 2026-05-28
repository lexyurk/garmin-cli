package training

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/lexyurk/garmin-cli/internal/client"
)

type RacePredictions struct {
	CalendarDate        string `json:"calendar_date,omitempty"`
	Time5KSeconds       *int   `json:"time_5k_seconds,omitempty"`
	Time10KSeconds      *int   `json:"time_10k_seconds,omitempty"`
	TimeHalfSeconds     *int   `json:"time_half_marathon_seconds,omitempty"`
	TimeMarathonSeconds *int   `json:"time_marathon_seconds,omitempty"`
}

type racePredictionsRaw struct {
	CalendarDate     string `json:"calendarDate"`
	Time5K           *int   `json:"time5K"`
	Time10K          *int   `json:"time10K"`
	TimeHalfMarathon *int   `json:"timeHalfMarathon"`
	TimeMarathon     *int   `json:"timeMarathon"`
}

// GetRacePredictions returns the latest race time predictions for a user.
func GetRacePredictions(ctx context.Context, c *client.Client, displayName string) (RacePredictions, error) {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return RacePredictions{}, fmt.Errorf("display name is required")
	}
	var raw racePredictionsRaw
	path := "/metrics-service/metrics/racepredictions/latest/" + url.PathEscape(displayName)
	if err := c.GetJSON(ctx, path, nil, &raw); err != nil {
		return RacePredictions{}, err
	}
	return RacePredictions{
		CalendarDate:        raw.CalendarDate,
		Time5KSeconds:       raw.Time5K,
		Time10KSeconds:      raw.Time10K,
		TimeHalfSeconds:     raw.TimeHalfMarathon,
		TimeMarathonSeconds: raw.TimeMarathon,
	}, nil
}
