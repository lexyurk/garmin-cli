// Package workouts manages Garmin Connect structured workouts.
package workouts

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/lexyurk/garmin-cli/internal/client"
)

type SportType struct {
	SportTypeID  int    `json:"sportTypeId"`
	SportTypeKey string `json:"sportTypeKey"`
}

type Summary struct {
	WorkoutID    int64   `json:"workout_id"`
	Name         string  `json:"name"`
	Sport        string  `json:"sport,omitempty"`
	DurationSecs float64 `json:"duration_seconds,omitempty"`
	DistanceM    float64 `json:"distance_meters,omitempty"`
	Description  string  `json:"description,omitempty"`
	Updated      string  `json:"updated,omitempty"`
}

type listItem struct {
	WorkoutID   int64     `json:"workoutId"`
	WorkoutName string    `json:"workoutName"`
	SportType   SportType `json:"sportType"`
	EstDuration float64   `json:"estimatedDurationInSecs"`
	EstDistance float64   `json:"estimatedDistanceInMeters"`
	Description string    `json:"description"`
	UpdatedDate string    `json:"updateDate"`
}

func (l listItem) toSummary() Summary {
	return Summary{
		WorkoutID:    l.WorkoutID,
		Name:         l.WorkoutName,
		Sport:        l.SportType.SportTypeKey,
		DurationSecs: l.EstDuration,
		DistanceM:    l.EstDistance,
		Description:  l.Description,
		Updated:      l.UpdatedDate,
	}
}

// List returns the user's saved workouts (most recent first).
func List(ctx context.Context, c *client.Client, limit int) ([]Summary, error) {
	if limit <= 0 {
		limit = 20
	}
	q := url.Values{
		"start": {"0"},
		"limit": {strconv.Itoa(limit)},
	}
	var raw []listItem
	if err := c.GetJSON(ctx, "/workout-service/workouts", q, &raw); err != nil {
		return nil, err
	}
	out := make([]Summary, 0, len(raw))
	for _, l := range raw {
		out = append(out, l.toSummary())
	}
	return out, nil
}

// GetRaw returns the full workout object as decoded JSON.
func GetRaw(ctx context.Context, c *client.Client, workoutID int64) (map[string]any, error) {
	var raw map[string]any
	if err := c.GetJSON(ctx, fmt.Sprintf("/workout-service/workout/%d", workoutID), nil, &raw); err != nil {
		return nil, err
	}
	return raw, nil
}

// SummarizeRaw extracts headline fields from a full workout object.
func SummarizeRaw(workoutID int64, raw map[string]any) Summary {
	s := Summary{WorkoutID: workoutID}
	if v, ok := raw["workoutName"].(string); ok {
		s.Name = v
	}
	if v, ok := raw["description"].(string); ok {
		s.Description = v
	}
	if st, ok := raw["sportType"].(map[string]any); ok {
		if k, ok := st["sportTypeKey"].(string); ok {
			s.Sport = k
		}
	}
	if v, ok := raw["estimatedDurationInSecs"].(float64); ok {
		s.DurationSecs = v
	}
	if v, ok := raw["estimatedDistanceInMeters"].(float64); ok {
		s.DistanceM = v
	}
	return s
}

// CountSteps returns the number of executable steps across all segments,
// expanding repeat groups by their iteration count.
func CountSteps(raw map[string]any) int {
	segs, _ := raw["workoutSegments"].([]any)
	total := 0
	for _, seg := range segs {
		segMap, _ := seg.(map[string]any)
		steps, _ := segMap["workoutSteps"].([]any)
		total += countStepList(steps)
	}
	return total
}

func countStepList(steps []any) int {
	total := 0
	for _, st := range steps {
		stMap, _ := st.(map[string]any)
		if child, ok := stMap["workoutSteps"].([]any); ok && len(child) > 0 {
			iters := 1
			if n, ok := stMap["numberOfIterations"].(float64); ok && n > 0 {
				iters = int(n)
			}
			total += iters * countStepList(child)
			continue
		}
		total++
	}
	return total
}
