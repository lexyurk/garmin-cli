// Package weight manages Garmin Connect weigh-ins and body composition.
package weight

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/lexyurk/garmin-cli/internal/client"
)

const tsLayout = "2006-01-02T15:04:05.000"

type WeighIn struct {
	Date       string   `json:"date,omitempty"`
	WeightKG   *float64 `json:"weight_kg,omitempty"`
	BMI        *float64 `json:"bmi,omitempty"`
	BodyFatPct *float64 `json:"body_fat_pct,omitempty"`
	BodyWater  *float64 `json:"body_water_pct,omitempty"`
	SamplePk   int64    `json:"sample_pk,omitempty"`
}

type rangeRaw struct {
	DateWeightList []weighInRaw `json:"dateWeightList"`
}

type weighInRaw struct {
	SamplePk     int64    `json:"samplePk"`
	CalendarDate string   `json:"calendarDate"`
	Weight       *float64 `json:"weight"` // grams
	BMI          *float64 `json:"bmi"`
	BodyFat      *float64 `json:"bodyFat"`
	BodyWater    *float64 `json:"bodyWater"`
}

func (w weighInRaw) toWeighIn() WeighIn {
	out := WeighIn{
		Date:       w.CalendarDate,
		BMI:        w.BMI,
		BodyFatPct: w.BodyFat,
		BodyWater:  w.BodyWater,
		SamplePk:   w.SamplePk,
	}
	if w.Weight != nil {
		kg := *w.Weight / 1000.0 // API returns grams
		out.WeightKG = &kg
	}
	return out
}

// List returns weigh-ins between startDate and endDate (inclusive, YYYY-MM-DD).
func List(ctx context.Context, c *client.Client, startDate, endDate string) ([]WeighIn, error) {
	if err := validateDate(startDate); err != nil {
		return nil, err
	}
	if err := validateDate(endDate); err != nil {
		return nil, err
	}
	q := url.Values{"startDate": {startDate}, "endDate": {endDate}}
	var raw rangeRaw
	if err := c.GetJSON(ctx, "/weight-service/weight/dateRange", q, &raw); err != nil {
		return nil, err
	}
	out := make([]WeighIn, 0, len(raw.DateWeightList))
	for _, w := range raw.DateWeightList {
		out = append(out, w.toWeighIn())
	}
	return out, nil
}

// Latest returns the most recent weigh-in within the past `days` days.
func Latest(ctx context.Context, c *client.Client, days int) (WeighIn, error) {
	if days <= 0 {
		days = 30
	}
	now := time.Now().In(time.Local)
	start := now.AddDate(0, 0, -(days - 1)).Format("2006-01-02")
	end := now.Format("2006-01-02")
	list, err := List(ctx, c, start, end)
	if err != nil {
		return WeighIn{}, err
	}
	if len(list) == 0 {
		return WeighIn{}, fmt.Errorf("no weigh-ins in the last %d days", days)
	}
	return list[len(list)-1], nil
}

// Add logs a weigh-in (kilograms) for the given date (default: now).
func Add(ctx context.Context, c *client.Client, weightKG float64, date string) error {
	if weightKG <= 0 {
		return fmt.Errorf("weight must be > 0")
	}
	local := time.Now().In(time.Local)
	if d := strings.TrimSpace(date); d != "" {
		t, err := time.ParseInLocation("2006-01-02", d, time.Local)
		if err != nil {
			return fmt.Errorf("invalid date %q (expected YYYY-MM-DD)", date)
		}
		// Use midday so the UTC conversion never shifts to an adjacent day.
		local = time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, time.Local)
	}

	payload := map[string]any{
		"dateTimestamp": local.Format(tsLayout),
		"gmtTimestamp":  local.UTC().Format(tsLayout),
		"unitKey":       "kg",
		"sourceType":    "MANUAL",
		"value":         weightKG,
	}
	return c.PostJSON(ctx, "/weight-service/user-weight", nil, payload, nil)
}

func validateDate(s string) error {
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return fmt.Errorf("invalid date %q (expected YYYY-MM-DD)", s)
	}
	return nil
}
