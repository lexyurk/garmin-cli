package health

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lexyurk/garmin-cli/internal/client"
)

// --- SpO2 (pulse ox) ---

type SpO2Summary struct {
	Date    string   `json:"date"`
	Average *float64 `json:"average,omitempty"`
	Lowest  *float64 `json:"lowest,omitempty"`
	Latest  *float64 `json:"latest,omitempty"`
}

type spo2Raw struct {
	CalendarDate string   `json:"calendarDate"`
	AverageSpO2  *float64 `json:"averageSpO2"`
	LowestSpO2   *float64 `json:"lowestSpO2"`
	LatestSpO2   *float64 `json:"latestSpO2"`
}

func GetSpO2(ctx context.Context, c *client.Client, date string) (SpO2Summary, error) {
	var raw spo2Raw
	if err := c.GetJSON(ctx, "/wellness-service/wellness/daily/spo2/"+date, nil, &raw); err != nil {
		return SpO2Summary{}, err
	}
	return SpO2Summary{
		Date:    orDate(raw.CalendarDate, date),
		Average: raw.AverageSpO2,
		Lowest:  raw.LowestSpO2,
		Latest:  raw.LatestSpO2,
	}, nil
}

// --- Respiration ---

type RespirationSummary struct {
	Date      string   `json:"date"`
	AvgWaking *float64 `json:"avg_waking,omitempty"`
	Highest   *float64 `json:"highest,omitempty"`
	Lowest    *float64 `json:"lowest,omitempty"`
}

type respirationRaw struct {
	CalendarDate              string   `json:"calendarDate"`
	AvgWakingRespirationValue *float64 `json:"avgWakingRespirationValue"`
	HighestRespirationValue   *float64 `json:"highestRespirationValue"`
	LowestRespirationValue    *float64 `json:"lowestRespirationValue"`
}

func GetRespiration(ctx context.Context, c *client.Client, date string) (RespirationSummary, error) {
	var raw respirationRaw
	if err := c.GetJSON(ctx, "/wellness-service/wellness/daily/respiration/"+date, nil, &raw); err != nil {
		return RespirationSummary{}, err
	}
	return RespirationSummary{
		Date:      orDate(raw.CalendarDate, date),
		AvgWaking: raw.AvgWakingRespirationValue,
		Highest:   raw.HighestRespirationValue,
		Lowest:    raw.LowestRespirationValue,
	}, nil
}

// --- Intensity minutes ---

type IntensityMinutesSummary struct {
	Date       string   `json:"date"`
	Moderate   *float64 `json:"moderate,omitempty"`
	Vigorous   *float64 `json:"vigorous,omitempty"`
	WeeklyGoal *float64 `json:"weekly_goal,omitempty"`
}

type intensityRaw struct {
	CalendarDate  string   `json:"calendarDate"`
	ModerateValue *float64 `json:"moderateValue"`
	VigorousValue *float64 `json:"vigorousValue"`
	WeeklyGoal    *float64 `json:"weeklyGoal"`
}

func GetIntensityMinutes(ctx context.Context, c *client.Client, date string) (IntensityMinutesSummary, error) {
	var raw intensityRaw
	if err := c.GetJSON(ctx, fmt.Sprintf("/wellness-service/wellness/daily/im/%s", date), nil, &raw); err != nil {
		return IntensityMinutesSummary{}, err
	}
	return IntensityMinutesSummary{
		Date:       orDate(raw.CalendarDate, date),
		Moderate:   raw.ModerateValue,
		Vigorous:   raw.VigorousValue,
		WeeklyGoal: raw.WeeklyGoal,
	}, nil
}

// --- Intraday stress ---

type StressPoint struct {
	TimestampMs int64 `json:"ts"`
	Level       int   `json:"level"`
}

type StressDetail struct {
	Date    string        `json:"date"`
	Average *int          `json:"average,omitempty"`
	Max     *int          `json:"max,omitempty"`
	Values  []StressPoint `json:"values"`
}

type stressDetailRaw struct {
	CalendarDate      string          `json:"calendarDate"`
	AvgStressLevel    *int            `json:"avgStressLevel"`
	MaxStressLevel    *int            `json:"maxStressLevel"`
	StressValuesArray [][]json.Number `json:"stressValuesArray"`
}

// GetStressDetail returns the intraday stress timeline (~3-minute samples).
// Negative levels are Garmin markers: -1 unmeasured, -2 activity in progress.
func GetStressDetail(ctx context.Context, c *client.Client, date string) (StressDetail, error) {
	var raw stressDetailRaw
	if err := c.GetJSON(ctx, "/wellness-service/wellness/dailyStress/"+date, nil, &raw); err != nil {
		return StressDetail{}, err
	}
	out := StressDetail{
		Date:    orDate(raw.CalendarDate, date),
		Average: raw.AvgStressLevel,
		Max:     raw.MaxStressLevel,
		Values:  make([]StressPoint, 0, len(raw.StressValuesArray)),
	}
	for _, pair := range raw.StressValuesArray {
		if len(pair) < 2 {
			continue
		}
		ts, err1 := pair[0].Int64()
		lvl, err2 := pair[1].Int64()
		if err1 != nil || err2 != nil {
			continue
		}
		out.Values = append(out.Values, StressPoint{TimestampMs: ts, Level: int(lvl)})
	}
	return out, nil
}

func orDate(calendarDate, fallback string) string {
	if calendarDate != "" {
		return calendarDate
	}
	return fallback
}
