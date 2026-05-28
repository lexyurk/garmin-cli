package activities

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/lexyurk/garmin-cli/internal/client"
)

type TypeInfo struct {
	TypeKey string `json:"typeKey"`
}

type ListItem struct {
	ActivityID     int64    `json:"activityId"`
	ActivityName   string   `json:"activityName"`
	StartTimeLocal string   `json:"startTimeLocal"`
	ActivityType   TypeInfo `json:"activityType"`
	Distance       float64  `json:"distance"`
	Duration       float64  `json:"duration"`
	Calories       int      `json:"calories"`
	AverageHR      int      `json:"averageHR"`
}

type Summary struct {
	ID              int64   `json:"id"`
	Date            string  `json:"date"`
	StartTimeLocal  string  `json:"start_time_local,omitempty"`
	Type            string  `json:"type"`
	Name            string  `json:"name"`
	DistanceMeters  float64 `json:"distance_meters"`
	DurationSeconds float64 `json:"duration_seconds"`
	Calories        int     `json:"calories,omitempty"`
	AvgHR           int     `json:"avg_hr,omitempty"`
}

func List(ctx context.Context, c *client.Client, limit int, after, before, activityType string) ([]Summary, error) {
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be > 0")
	}

	if after != "" {
		d, err := parseDate(after)
		if err != nil {
			return nil, err
		}
		after = d
	}
	if before != "" {
		d, err := parseDate(before)
		if err != nil {
			return nil, err
		}
		before = d
	}
	if after != "" && before != "" && before < after {
		return nil, fmt.Errorf("--before (%s) is before --after (%s)", before, after)
	}

	var out []Summary
	start := 0
	pageSize := 50
	if limit < pageSize {
		pageSize = limit
	}

	for len(out) < limit {
		var page []ListItem
		q := url.Values{
			"limit": {strconv.Itoa(pageSize)},
			"start": {strconv.Itoa(start)},
		}
		if err := c.GetJSON(ctx, "/activitylist-service/activities/search/activities", q, &page); err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}

		for _, item := range page {
			s := item.ToSummary()
			if !passesFilters(s, after, before, activityType) {
				continue
			}
			out = append(out, s)
			if len(out) >= limit {
				break
			}
		}

		if after != "" {
			oldest := page[len(page)-1].startDate()
			if oldest != "" && oldest < after {
				break
			}
		}

		start += len(page)
		if len(page) < pageSize {
			break
		}
	}

	return out, nil
}

// ListByGear returns activities recorded with a given gear item (most recent first).
func ListByGear(ctx context.Context, c *client.Client, gearUUID string, limit int) ([]Summary, error) {
	gearUUID = strings.TrimSpace(gearUUID)
	if gearUUID == "" {
		return nil, fmt.Errorf("gear uuid is required")
	}
	if limit <= 0 {
		limit = 20
	}
	q := url.Values{
		"start": {"0"},
		"limit": {strconv.Itoa(limit)},
	}
	var page []ListItem
	if err := c.GetJSON(ctx, fmt.Sprintf("/activitylist-service/activities/%s/gear", url.PathEscape(gearUUID)), q, &page); err != nil {
		return nil, err
	}
	out := make([]Summary, 0, len(page))
	for _, item := range page {
		out = append(out, item.ToSummary())
	}
	return out, nil
}

// Latest returns the most recent activity.
func Latest(ctx context.Context, c *client.Client) (Summary, error) {
	list, err := List(ctx, c, 1, "", "", "")
	if err != nil {
		return Summary{}, err
	}
	if len(list) == 0 {
		return Summary{}, fmt.Errorf("no activities found")
	}
	return list[0], nil
}

func GetRaw(ctx context.Context, c *client.Client, activityID int64) (map[string]any, error) {
	var raw map[string]any
	if err := c.GetJSON(ctx, fmt.Sprintf("/activity-service/activity/%d", activityID), nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

func (a ListItem) startDate() string {
	if len(a.StartTimeLocal) >= 10 {
		return a.StartTimeLocal[:10]
	}
	return ""
}

func (a ListItem) ToSummary() Summary {
	return Summary{
		ID:              a.ActivityID,
		Date:            a.startDate(),
		StartTimeLocal:  a.StartTimeLocal,
		Type:            a.ActivityType.TypeKey,
		Name:            a.ActivityName,
		DistanceMeters:  a.Distance,
		DurationSeconds: a.Duration,
		Calories:        a.Calories,
		AvgHR:           a.AverageHR,
	}
}

func passesFilters(a Summary, after, before, activityType string) bool {
	if after != "" && a.Date != "" && a.Date < after {
		return false
	}
	if before != "" && a.Date != "" && a.Date > before {
		return false
	}
	if strings.TrimSpace(activityType) != "" {
		t := strings.ToLower(strings.TrimSpace(activityType))
		typ := strings.ToLower(a.Type)
		if typ != t && !strings.Contains(typ, t) {
			return false
		}
	}
	return true
}

func parseDate(s string) (string, error) {
	t, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		return "", fmt.Errorf("invalid date %q (expected YYYY-MM-DD)", s)
	}
	return t.Format("2006-01-02"), nil
}
