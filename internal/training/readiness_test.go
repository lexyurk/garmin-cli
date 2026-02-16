package training

import (
	"encoding/json"
	"testing"
)

func TestGetReadiness_SummarizeMostRecentTimestamp(t *testing.T) {
	raw := []byte(`[
	  {"calendarDate":"2026-02-16","timestamp":"2026-02-16T07:00:00.0","level":"LOW","score":30},
	  {"calendarDate":"2026-02-16","timestamp":"2026-02-16T09:00:00.0","level":"HIGH","score":80}
	]`)

	var entries []ReadinessEntry
	if err := json.Unmarshal(raw, &entries); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	s := summarizeReadiness("fallback", entries)
	if s.Date != "2026-02-16" {
		t.Fatalf("date: %q", s.Date)
	}
	if s.Level != "HIGH" {
		t.Fatalf("level: %q", s.Level)
	}
	if s.Score == nil || *s.Score != 80 {
		t.Fatalf("score: %#v", s.Score)
	}
}

