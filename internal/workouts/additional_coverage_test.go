package workouts

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/lexyurk/garmin-cli/internal/auth"
)

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestBuildPayload_DescriptionAndProgrammaticStepErrors(t *testing.T) {
	payload, err := BuildPayload("lap workout", " description ", "cycling", []Step{{Type: "interval", LapButton: true}})
	if err != nil || payload["description"] != " description " {
		t.Fatalf("payload=%#v err=%v", payload, err)
	}

	for _, tc := range []struct {
		name  string
		steps []Step
	}{
		{"zero repeat", []Step{{Kind: RepeatStep, Iterations: 0, Children: []Step{{Type: "warmup", DurationSec: 1}}}}},
		{"empty repeat", []Step{{Kind: RepeatStep, Iterations: 2}}},
		{"bad nested child", []Step{{Kind: RepeatStep, Iterations: 2, Children: []Step{{Type: "unknown", DurationSec: 1}}}}},
		{"unknown type", []Step{{Type: "unknown", DurationSec: 1}}},
		{"repeat as executable", []Step{{Type: "repeat", DurationSec: 1}}},
		{"missing condition", []Step{{Type: "warmup"}}},
		{"zero parsed amount", []Step{{Type: "warmup", DurationSec: 0}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := BuildPayload("x", "", "running", tc.steps); err == nil {
				t.Fatal("expected build error")
			}
		})
	}
}

func TestParseSteps_EmptyRepeatAndMalformedAmounts(t *testing.T) {
	for _, spec := range []string{
		"2x( ; )",
		"2x(warmup)",
		"warmup nope-min",
		"warmup nope-sec",
		"warmup nope-km",
		"warmup nope-s",
		"warmup nope-m",
	} {
		if _, err := ParseSteps([]string{spec}); err == nil {
			t.Fatalf("expected %q to fail", spec)
		}
	}
}

func TestWorkoutAPIs_DefaultsFallbacksAndErrors(t *testing.T) {
	bad := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	})
	if _, err := List(context.Background(), bad, 0); err == nil {
		t.Fatal("expected list error after defaulting limit")
	}
	if _, err := GetRaw(context.Background(), bad, 1); err == nil {
		t.Fatal("expected get error")
	}
	if _, err := Schedule(context.Background(), bad, 1, "2026-06-01"); err == nil {
		t.Fatal("expected schedule error")
	}
	name := "new"
	if _, err := Update(context.Background(), bad, 1, UpdateOptions{Name: &name}); err == nil {
		t.Fatal("expected update get error")
	}

	create := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{}`)
	})
	s, err := Create(context.Background(), create, map[string]any{"workoutName": "payload name"})
	if err != nil || s.Name != "payload name" {
		t.Fatalf("summary=%#v err=%v", s, err)
	}
	if _, err := Create(context.Background(), bad, map[string]any{}); err == nil {
		t.Fatal("expected create error")
	}

	putBad := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_, _ = io.WriteString(w, `{"workoutName":"old"}`)
			return
		}
		http.Error(w, "bad", http.StatusBadRequest)
	})
	description := "new description"
	if _, err := Update(context.Background(), putBad, 1, UpdateOptions{Description: &description}); err == nil {
		t.Fatal("expected update put error")
	}
}

func TestCountSteps_DefaultRepeatAndLooseShapes(t *testing.T) {
	raw := map[string]any{"workoutSegments": []any{
		map[string]any{"workoutSteps": []any{
			map[string]any{"workoutSteps": []any{map[string]any{"stepType": "x"}}},
			"not-a-map",
		}},
		"not-a-map",
	}}
	if got := CountSteps(raw); got != 2 {
		t.Fatalf("CountSteps=%d", got)
	}
}

func TestExportFIT_StatusAndWriterErrors(t *testing.T) {
	for _, status := range []int{http.StatusUnauthorized, http.StatusBadRequest} {
		c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "rejected", status)
		})
		err := ExportFIT(context.Background(), c, 1, io.Discard)
		if err == nil {
			t.Fatalf("expected status %d error", status)
		}
		if status == http.StatusUnauthorized && !errors.Is(err, auth.ErrNotAuthenticated) {
			t.Fatalf("expected auth error, got %v", err)
		}
	}
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "FIT")
	})
	if err := ExportFIT(context.Background(), c, 1, errorWriter{}); err == nil {
		t.Fatal("expected writer error")
	}
}

func TestUnschedule_PropagatesError(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad", http.StatusBadRequest)
	})
	if err := Unschedule(context.Background(), c, 1); err == nil || !strings.Contains(err.Error(), "400") {
		t.Fatalf("unexpected error: %v", err)
	}
}
