package training

import (
	"encoding/json"
	"testing"
)

func TestHRVResponse_ToSummary(t *testing.T) {
	raw := []byte(`{
	  "hrvSummary": {
	    "calendarDate": "2026-02-16",
	    "weeklyAvg": 40,
	    "lastNightAvg": 42,
	    "status": "BALANCED",
	    "baseline": {"lowUpper": 36, "balancedUpper": 52}
	  }
	}`)

	var resp HRVResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	s := resp.ToSummary("fallback")
	if s.Date != "2026-02-16" {
		t.Fatalf("date: %q", s.Date)
	}
	if s.Status != "BALANCED" {
		t.Fatalf("status: %q", s.Status)
	}
	if s.BaselineLowUpper == nil || *s.BaselineLowUpper != 36 {
		t.Fatalf("baseline low: %#v", s.BaselineLowUpper)
	}
}

