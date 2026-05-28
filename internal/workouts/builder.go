package workouts

import (
	"context"
	"fmt"
	"strings"

	"github.com/lexyurk/garmin-cli/internal/client"
)

// Verified Garmin workout-service type IDs (reverse-engineered; see builder docs).
var (
	sportTypeIDs = map[string]int{
		"running": 1,
		"cycling": 2,
	}
	stepTypeIDs = map[string]int{
		"warmup":   1,
		"cooldown": 2,
		"interval": 3,
		"recovery": 4,
		"rest":     5,
		"repeat":   6,
		"other":    8,
	}
)

const (
	condLapButton  = 1
	condTime       = 2
	condDistance   = 3
	condIterations = 7
)

// StepKind distinguishes an executable step from a repeat block.
type StepKind int

const (
	ExecutableStep StepKind = iota
	RepeatStep
)

// Step is a node in a workout's step tree.
type Step struct {
	Kind        StepKind
	Type        string // step type key for executable steps (warmup, interval, ...)
	DurationSec float64
	DistanceM   float64
	LapButton   bool
	Iterations  int
	Children    []Step
}

// BuildPayload assembles a workout-service POST body from a step tree.
func BuildPayload(name, description, sportKey string, steps []Step) (map[string]any, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("workout name is required")
	}
	sportKey = strings.ToLower(strings.TrimSpace(sportKey))
	sid, ok := sportTypeIDs[sportKey]
	if !ok {
		return nil, fmt.Errorf("unsupported sport %q (supported: running, cycling)", sportKey)
	}
	if len(steps) == 0 {
		return nil, fmt.Errorf("at least one step is required")
	}

	order := 1
	built, err := buildStepList(steps, &order)
	if err != nil {
		return nil, err
	}

	sport := map[string]any{"sportTypeId": sid, "sportTypeKey": sportKey}
	payload := map[string]any{
		"workoutName": name,
		"sportType":   sport,
		"workoutSegments": []any{
			map[string]any{
				"segmentOrder": 1,
				"sportType":    sport,
				"workoutSteps": built,
			},
		},
	}
	if strings.TrimSpace(description) != "" {
		payload["description"] = description
	}
	return payload, nil
}

func buildStepList(steps []Step, order *int) ([]any, error) {
	out := make([]any, 0, len(steps))
	for _, s := range steps {
		if s.Kind == RepeatStep {
			if s.Iterations < 1 {
				return nil, fmt.Errorf("repeat must iterate at least once")
			}
			groupOrder := *order
			*order++
			children, err := buildStepList(s.Children, order)
			if err != nil {
				return nil, err
			}
			if len(children) == 0 {
				return nil, fmt.Errorf("repeat block has no steps")
			}
			out = append(out, map[string]any{
				"type":               "RepeatGroupDTO",
				"stepOrder":          groupOrder,
				"stepType":           map[string]any{"stepTypeId": stepTypeIDs["repeat"], "stepTypeKey": "repeat"},
				"numberOfIterations": s.Iterations,
				"smartRepeat":        false,
				"endCondition":       map[string]any{"conditionTypeId": condIterations, "conditionTypeKey": "iterations"},
				"endConditionValue":  float64(s.Iterations),
				"workoutSteps":       children,
			})
			continue
		}

		js, err := executableStepJSON(*order, s)
		if err != nil {
			return nil, err
		}
		*order++
		out = append(out, js)
	}
	return out, nil
}

func executableStepJSON(order int, s Step) (map[string]any, error) {
	typeKey := strings.ToLower(strings.TrimSpace(s.Type))
	stid, ok := stepTypeIDs[typeKey]
	if !ok || typeKey == "repeat" {
		return nil, fmt.Errorf("unknown step type %q", s.Type)
	}

	step := map[string]any{
		"type":       "ExecutableStepDTO",
		"stepOrder":  order,
		"stepType":   map[string]any{"stepTypeId": stid, "stepTypeKey": typeKey},
		"targetType": map[string]any{"workoutTargetTypeId": 1, "workoutTargetTypeKey": "no.target"},
	}

	switch {
	case s.LapButton:
		step["endCondition"] = map[string]any{"conditionTypeId": condLapButton, "conditionTypeKey": "lap.button"}
	case s.DistanceM > 0:
		step["endCondition"] = map[string]any{"conditionTypeId": condDistance, "conditionTypeKey": "distance"}
		step["endConditionValue"] = s.DistanceM
	case s.DurationSec > 0:
		step["endCondition"] = map[string]any{"conditionTypeId": condTime, "conditionTypeKey": "time"}
		step["endConditionValue"] = s.DurationSec
	default:
		return nil, fmt.Errorf("step %q needs a duration, distance, or 'lap'", typeKey)
	}
	return step, nil
}

// Create posts a workout and returns its summary (including the new workout id).
func Create(ctx context.Context, c *client.Client, payload map[string]any) (Summary, error) {
	var out map[string]any
	if err := c.PostJSON(ctx, "/workout-service/workout", nil, payload, &out); err != nil {
		return Summary{}, err
	}
	var id int64
	if v, ok := out["workoutId"].(float64); ok {
		id = int64(v)
	}
	s := SummarizeRaw(id, out)
	if s.Name == "" {
		if n, ok := payload["workoutName"].(string); ok {
			s.Name = n
		}
	}
	return s, nil
}
