package workouts

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestParseSteps_SimpleAndRepeat(t *testing.T) {
	steps, err := ParseSteps([]string{
		"warmup 10min",
		"4x(interval 800m; recovery 2min)",
		"cooldown 5min",
	})
	if err != nil {
		t.Fatalf("ParseSteps: %v", err)
	}
	if len(steps) != 3 {
		t.Fatalf("expected 3 top-level steps, got %d", len(steps))
	}
	if steps[0].Type != "warmup" || steps[0].DurationSec != 600 {
		t.Fatalf("warmup: %#v", steps[0])
	}
	if steps[1].Kind != RepeatStep || steps[1].Iterations != 4 || len(steps[1].Children) != 2 {
		t.Fatalf("repeat: %#v", steps[1])
	}
	if steps[1].Children[0].DistanceM != 800 {
		t.Fatalf("interval distance: %#v", steps[1].Children[0])
	}
	if steps[1].Children[1].DurationSec != 120 {
		t.Fatalf("recovery time: %#v", steps[1].Children[1])
	}
}

func TestParseSteps_LapAndUnits(t *testing.T) {
	steps, err := ParseSteps([]string{"interval lap", "interval 1.5km", "rest 45s"})
	if err != nil {
		t.Fatalf("ParseSteps: %v", err)
	}
	if !steps[0].LapButton {
		t.Fatalf("expected lap button: %#v", steps[0])
	}
	if steps[1].DistanceM != 1500 {
		t.Fatalf("km parse: %#v", steps[1])
	}
	if steps[2].DurationSec != 45 {
		t.Fatalf("s parse: %#v", steps[2])
	}
}

func TestParseSteps_Errors(t *testing.T) {
	for _, bad := range []string{"warmup", "frobnicate 10min", "warmup 10furlongs", "0x(interval 800m)"} {
		if _, err := ParseSteps([]string{bad}); err == nil {
			t.Fatalf("expected error for %q", bad)
		}
	}
}

func TestBuildPayload_StructureAndTypeIDs(t *testing.T) {
	steps, err := ParseSteps([]string{
		"warmup 10min",
		"4x(interval 800m; recovery 2min)",
		"cooldown 5min",
	})
	if err != nil {
		t.Fatalf("ParseSteps: %v", err)
	}
	payload, err := BuildPayload("4x800m", "speed", "running", steps)
	if err != nil {
		t.Fatalf("BuildPayload: %v", err)
	}

	// Round-trip through JSON so we inspect it the way the API would receive it.
	b, _ := json.Marshal(payload)
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("marshal round-trip: %v", err)
	}

	sport := got["sportType"].(map[string]any)
	if sport["sportTypeId"].(float64) != 1 || sport["sportTypeKey"] != "running" {
		t.Fatalf("sportType: %#v", sport)
	}

	seg := got["workoutSegments"].([]any)[0].(map[string]any)
	stepList := seg["workoutSteps"].([]any)
	if len(stepList) != 3 {
		t.Fatalf("expected 3 segment steps, got %d", len(stepList))
	}

	warmup := stepList[0].(map[string]any)
	if warmup["type"] != "ExecutableStepDTO" || warmup["stepOrder"].(float64) != 1 {
		t.Fatalf("warmup step: %#v", warmup)
	}
	if warmup["endConditionValue"].(float64) != 600 {
		t.Fatalf("warmup duration: %#v", warmup["endConditionValue"])
	}

	repeat := stepList[1].(map[string]any)
	if repeat["type"] != "RepeatGroupDTO" || repeat["numberOfIterations"].(float64) != 4 {
		t.Fatalf("repeat: %#v", repeat)
	}
	if repeat["stepOrder"].(float64) != 2 {
		t.Fatalf("repeat order: %#v", repeat["stepOrder"])
	}
	children := repeat["workoutSteps"].([]any)
	interval := children[0].(map[string]any)
	if interval["stepOrder"].(float64) != 3 {
		t.Fatalf("interval order should be 3 (flattened): %#v", interval["stepOrder"])
	}
	ec := interval["endCondition"].(map[string]any)
	if ec["conditionTypeId"].(float64) != 3 || ec["conditionTypeKey"] != "distance" {
		t.Fatalf("interval end condition (distance=3): %#v", ec)
	}
	if interval["endConditionValue"].(float64) != 800 {
		t.Fatalf("interval distance: %#v", interval["endConditionValue"])
	}

	cooldown := stepList[2].(map[string]any)
	if cooldown["stepOrder"].(float64) != 5 {
		t.Fatalf("cooldown order should be 5: %#v", cooldown["stepOrder"])
	}
}

func TestBuildPayload_Validation(t *testing.T) {
	steps, _ := ParseSteps([]string{"warmup 10min"})
	if _, err := BuildPayload("", "", "running", steps); err == nil {
		t.Fatalf("expected error for empty name")
	}
	if _, err := BuildPayload("x", "", "swimming", steps); err == nil {
		t.Fatalf("expected error for unsupported sport")
	}
	if _, err := BuildPayload("x", "", "running", nil); err == nil {
		t.Fatalf("expected error for no steps")
	}
}

func TestCreate_PostsPayloadAndReturnsID(t *testing.T) {
	var path string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"workoutId":424242,"workoutName":"4x800m","sportType":{"sportTypeKey":"running"}}`))
	})

	s, err := Create(context.Background(), c, map[string]any{"workoutName": "4x800m"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if path != "/workout-service/workout" {
		t.Fatalf("path: %s", path)
	}
	if s.WorkoutID != 424242 || s.Name != "4x800m" {
		t.Fatalf("unexpected summary: %#v", s)
	}
}
