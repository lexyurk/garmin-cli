// Package weight manages Garmin Connect weigh-ins and body composition.
package weight

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
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

	// takenAt orders weigh-ins that share a calendar day. It is not serialized.
	takenAt time.Time
}

type rangeRaw struct {
	DateWeightList []weighInRaw `json:"dateWeightList"`
}

type weighInRaw struct {
	SamplePk     int64    `json:"samplePk"`
	CalendarDate string   `json:"calendarDate"`
	Date         any      `json:"date"`         // epoch ms, sometimes a string
	TimestampGMT any      `json:"timestampGMT"` // epoch ms or a timestamp string
	Weight       *float64 `json:"weight"`       // grams
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
		takenAt:    w.instant(),
	}
	if w.Weight != nil {
		kg := *w.Weight / 1000.0 // API returns grams
		out.WeightKG = &kg
	}
	return out
}

func (w weighInRaw) instant() time.Time {
	if t, ok := parseInstant(w.TimestampGMT); ok {
		return t
	}
	if t, ok := parseInstant(w.Date); ok {
		return t
	}
	return time.Time{}
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

// Latest returns the weigh-in with the newest calendar date in the past
// `days` days, using the measurement time to break ties on that day.
// dateRange order is not significant (it is often newest-first).
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
	best := list[0]
	for _, w := range list[1:] {
		if weighInNewer(w, best) {
			best = w
		}
	}
	return best, nil
}

// weighInNewer reports whether a is more recent than b.
func weighInNewer(a, b WeighIn) bool {
	if a.Date != b.Date {
		return a.Date > b.Date
	}
	return a.takenAt.After(b.takenAt)
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

// parseInstant accepts a Garmin timestamp: epoch milliseconds, or a
// timestamp string (RFC3339 or Connect's no-zone form).
func parseInstant(v any) (time.Time, bool) {
	switch t := v.(type) {
	case float64:
		return epochMilli(int64(t))
	case string:
		s := strings.TrimSpace(t)
		if s == "" {
			return time.Time{}, false
		}
		if ms, err := strconv.ParseInt(s, 10, 64); err == nil {
			return epochMilli(ms)
		}
		for _, layout := range []string{
			time.RFC3339Nano,
			"2006-01-02T15:04:05.000",
			"2006-01-02T15:04:05.0",
			"2006-01-02T15:04:05",
		} {
			if parsed, err := time.Parse(layout, s); err == nil {
				return parsed, true
			}
		}
	}
	return time.Time{}, false
}

func epochMilli(ms int64) (time.Time, bool) {
	if ms <= 0 {
		return time.Time{}, false
	}
	return time.UnixMilli(ms).UTC(), true
}
