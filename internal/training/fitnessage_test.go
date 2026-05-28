package training

import (
	"encoding/json"
	"testing"
)

func TestSummarizeFitnessAge(t *testing.T) {
	raw := []byte(`{
	  "chronologicalAge": 38,
	  "fitnessAge": 32.5,
	  "achievableFitnessAge": 30,
	  "previousFitnessAge": 33,
	  "components": {
	    "rhr": {"value": 52},
	    "bmi": {"value": 23.4},
	    "vigorousDays": {"value": 3},
	    "noValue": {}
	  }
	}`)

	var r fitnessAgeRaw
	if err := json.Unmarshal(raw, &r); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	s := summarizeFitnessAge("2026-05-28", r)
	if s.Date != "2026-05-28" {
		t.Fatalf("date: %q", s.Date)
	}
	if s.FitnessAge == nil || *s.FitnessAge != 32.5 {
		t.Fatalf("fitness_age: %#v", s.FitnessAge)
	}
	if s.ChronologicalAge == nil || *s.ChronologicalAge != 38 {
		t.Fatalf("chronological_age: %#v", s.ChronologicalAge)
	}
	if s.AchievableAge == nil || *s.AchievableAge != 30 {
		t.Fatalf("achievable: %#v", s.AchievableAge)
	}
	if got := s.Components["rhr"]; got != 52 {
		t.Fatalf("component rhr: %v", got)
	}
	if got := s.Components["bmi"]; got != 23.4 {
		t.Fatalf("component bmi: %v", got)
	}
	if _, ok := s.Components["noValue"]; ok {
		t.Fatalf("expected component without value to be skipped")
	}
}
