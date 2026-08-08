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
	    "sleepStartTimestampGMT": 1771193880000,
	    "sleepStartTimestampLocal": 1771197480000,
	    "sleepEndTimestampGMT": 1771221300000,
	    "sleepEndTimestampLocal": 1771224900000,
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
	if s.SleepStart == nil || *s.SleepStart != "2026-02-15T23:18:00+01:00" {
		t.Fatalf("unexpected sleep start: %#v", s.SleepStart)
	}
	if s.SleepEnd == nil || *s.SleepEnd != "2026-02-16T06:55:00+01:00" {
		t.Fatalf("unexpected sleep end: %#v", s.SleepEnd)
	}
}

func TestGarminLocalTimestamp_UsesTheOffsetEncodedByGarmin(t *testing.T) {
	gmt := int64(1782860400000)
	local := int64(1782867600000)

	got := garminLocalTimestamp(&gmt, &local)
	if got == nil || *got != "2026-07-01T01:00:00+02:00" {
		t.Fatalf("unexpected summer timestamp: %#v", got)
	}
}

func TestGarminLocalTimestamp_RequiresAnAbsoluteTimestamp(t *testing.T) {
	local := int64(1782867600000)
	if got := garminLocalTimestamp(nil, &local); got != nil {
		t.Fatalf("expected nil without GMT timestamp, got %#v", got)
	}
}
