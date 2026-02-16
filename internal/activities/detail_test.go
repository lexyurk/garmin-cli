package activities

import "testing"

func TestExtractSplits(t *testing.T) {
	raw := map[string]any{
		"splitSummaries": []any{
			map[string]any{"distance": 1000.0, "duration": 300.0, "averageHR": 150.0, "maxHR": 170.0},
			map[string]any{"distance": 1000.0, "duration": 290.0},
		},
	}

	splits := ExtractSplits(raw)
	if len(splits) != 2 {
		t.Fatalf("expected 2 splits, got %d", len(splits))
	}
	if splits[0].DurationSeconds != 300.0 {
		t.Fatalf("unexpected duration: %#v", splits[0])
	}
	if splits[0].AverageHR != 150 {
		t.Fatalf("unexpected avg hr: %#v", splits[0])
	}
}

