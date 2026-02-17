package activities

import "testing"

func TestSummarizeDetail_AndIntFromAny(t *testing.T) {
	raw := map[string]any{
		"activityName":         "Morning Run",
		"startTimeLocal":       "2026-02-16 07:00:00",
		"activityType":         map[string]any{"typeKey": "running"},
		"distance":             float64(5000),
		"duration":             int64(1500),
		"calories":             float64(321),
		"averageHR":            float64(140),
		"maxHR":                int(180),
		"elevationGain":        float64(12.5),
		"vO2MaxValue":          float64(52.3),
		"activityTrainingLoad": float64(123.4),
	}

	s := SummarizeDetail(123, raw)
	if s.ID != 123 || s.Name != "Morning Run" || s.Type != "running" {
		t.Fatalf("unexpected summary: %#v", s)
	}
	if s.DurationSeconds != 1500 || s.DistanceMeters != 5000 {
		t.Fatalf("unexpected duration/distance: %#v", s)
	}
	if s.Calories != 321 || s.AvgHR != 140 || s.MaxHR != 180 {
		t.Fatalf("unexpected HR/calories: %#v", s)
	}

	if got := intFromAny("nope"); got != 0 {
		t.Fatalf("expected 0 for unsupported type, got %d", got)
	}
}

func TestExtractSplits_EmptyOrMissing(t *testing.T) {
	if got := ExtractSplits(map[string]any{}); got != nil {
		t.Fatalf("expected nil for missing key, got %#v", got)
	}
	if got := ExtractSplits(map[string]any{"splitSummaries": []any{}}); got != nil {
		t.Fatalf("expected nil for empty splits, got %#v", got)
	}
}
