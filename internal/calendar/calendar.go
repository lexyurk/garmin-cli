// Package calendar reads the Garmin Connect training calendar.
package calendar

import (
	"context"
	"fmt"
	"strings"

	"github.com/lexyurk/garmin-cli/internal/client"
)

type Item struct {
	Date       string `json:"date,omitempty"`
	Type       string `json:"item_type,omitempty"`
	Title      string `json:"title,omitempty"`
	WorkoutID  int64  `json:"workout_id,omitempty"`
	ActivityID int64  `json:"activity_id,omitempty"`
}

type calItemRaw struct {
	Date         string `json:"date"`
	CalendarDate string `json:"calendarDate"`
	ItemType     string `json:"itemType"`
	Title        string `json:"title"`
	WorkoutID    int64  `json:"workoutId"`
	ActivityID   int64  `json:"activityId"`
}

type monthRaw struct {
	CalendarItems []calItemRaw `json:"calendarItems"`
}

// Month returns calendar items for the given year and 1-based month.
//
// The calendar-service indexes months 0-based (January = 0); this function
// accepts a natural 1-based month and converts internally.
func Month(ctx context.Context, c *client.Client, year, month int) ([]Item, error) {
	if month < 1 || month > 12 {
		return nil, fmt.Errorf("month must be 1-12, got %d", month)
	}
	path := fmt.Sprintf("/calendar-service/year/%d/month/%d", year, month-1)
	var raw monthRaw
	if err := c.GetJSON(ctx, path, nil, &raw); err != nil {
		return nil, err
	}
	out := make([]Item, 0, len(raw.CalendarItems))
	for _, it := range raw.CalendarItems {
		date := it.Date
		if date == "" {
			date = it.CalendarDate
		}
		out = append(out, Item{
			Date:       date,
			Type:       it.ItemType,
			Title:      it.Title,
			WorkoutID:  it.WorkoutID,
			ActivityID: it.ActivityID,
		})
	}
	return out, nil
}

// FilterByType returns items whose itemType matches the filter (case-insensitive).
// An empty filter returns all items.
func FilterByType(items []Item, itemType string) []Item {
	itemType = strings.ToLower(strings.TrimSpace(itemType))
	if itemType == "" {
		return items
	}
	out := make([]Item, 0, len(items))
	for _, it := range items {
		if strings.Contains(strings.ToLower(it.Type), itemType) {
			out = append(out, it)
		}
	}
	return out
}
