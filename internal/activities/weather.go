package activities

import (
	"context"
	"fmt"

	"github.com/lexyurk/garmin-cli/internal/client"
)

// Weather summarizes the weather recorded during an activity.
type Weather struct {
	ActivityID    int64    `json:"activity_id"`
	Temp          *float64 `json:"temp,omitempty"`
	ApparentTemp  *float64 `json:"apparent_temp,omitempty"`
	DewPoint      *float64 `json:"dew_point,omitempty"`
	Humidity      *int     `json:"relative_humidity,omitempty"`
	WindSpeed     *float64 `json:"wind_speed,omitempty"`
	WindDirection string   `json:"wind_direction,omitempty"`
	Description   string   `json:"description,omitempty"`
}

type weatherRaw struct {
	Temp                      *float64 `json:"temp"`
	ApparentTemp              *float64 `json:"apparentTemp"`
	DewPoint                  *float64 `json:"dewPoint"`
	RelativeHumidity          *int     `json:"relativeHumidity"`
	WindSpeed                 *float64 `json:"windSpeed"`
	WindDirectionCompassPoint string   `json:"windDirectionCompassPoint"`
	WeatherTypeDTO            struct {
		Desc string `json:"desc"`
	} `json:"weatherTypeDTO"`
}

// GetWeatherRaw returns the raw weather payload for an activity.
func GetWeatherRaw(ctx context.Context, c *client.Client, activityID int64) (map[string]any, error) {
	var raw map[string]any
	if err := c.GetJSON(ctx, fmt.Sprintf("/activity-service/activity/%d/weather", activityID), nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// GetWeather returns a typed weather summary for an activity.
func GetWeather(ctx context.Context, c *client.Client, activityID int64) (Weather, error) {
	var raw weatherRaw
	if err := c.GetJSON(ctx, fmt.Sprintf("/activity-service/activity/%d/weather", activityID), nil, &raw); err != nil {
		return Weather{}, err
	}
	return Weather{
		ActivityID:    activityID,
		Temp:          raw.Temp,
		ApparentTemp:  raw.ApparentTemp,
		DewPoint:      raw.DewPoint,
		Humidity:      raw.RelativeHumidity,
		WindSpeed:     raw.WindSpeed,
		WindDirection: raw.WindDirectionCompassPoint,
		Description:   raw.WeatherTypeDTO.Desc,
	}, nil
}
