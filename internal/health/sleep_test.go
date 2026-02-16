package health

import (
	"encoding/json"
	"testing"
)

func TestSleepDailyResponse_ToSummary(t *testing.T) {
	raw := []byte(`{
	  "dailySleepDTO": {
	    "calendarDate": "2026-02-15",
	    "sleepTimeSeconds": 28800,
	    "deepSleepSeconds": 5400,
	    "lightSleepSeconds": 15000,
	    "remSleepSeconds": 7200,
	    "awakeSleepSeconds": 1200,
	    "averageSpO2Value": 94.0,
	    "averageRespirationValue": 13.5,
	    "sleepScores": { "overall": { "value": 82 } }
	  }
	}`)

	var resp SleepDailyResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	s := resp.ToSummary("fallback")
	if s.Date != "2026-02-15" {
		t.Fatalf("unexpected date: %q", s.Date)
	}
	if s.Score == nil || *s.Score != 82 {
		t.Fatalf("unexpected score: %#v", s.Score)
	}
	if s.TotalSleepSeconds == nil || *s.TotalSleepSeconds != 28800 {
		t.Fatalf("unexpected total: %#v", s.TotalSleepSeconds)
	}
	if s.DeepSeconds == nil || *s.DeepSeconds != 5400 {
		t.Fatalf("unexpected deep: %#v", s.DeepSeconds)
	}
	if s.AvgSpO2 == nil || *s.AvgSpO2 != 94.0 {
		t.Fatalf("unexpected spO2: %#v", s.AvgSpO2)
	}
}
