package health

import (
	"encoding/json"
	"testing"
)

func TestDailySummary_Unmarshal(t *testing.T) {
	raw := []byte(`{
	  "calendarDate": "2026-02-16",
	  "totalSteps": 1234,
	  "dailyStepGoal": 8000,
	  "totalDistanceMeters": 950,
	  "minHeartRate": 45,
	  "maxHeartRate": 155,
	  "restingHeartRate": 52,
	  "averageStressLevel": 23,
	  "maxStressLevel": 86,
	  "stressQualifier": "LOW",
	  "bodyBatteryChargedValue": 50,
	  "bodyBatteryDrainedValue": 60,
	  "bodyBatteryHighestValue": 92,
	  "bodyBatteryLowestValue": 35,
	  "bodyBatteryMostRecentValue": 38
	}`)

	var d DailySummary
	if err := json.Unmarshal(raw, &d); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if d.CalendarDate != "2026-02-16" {
		t.Fatalf("calendarDate: %q", d.CalendarDate)
	}
	if d.TotalSteps == nil || *d.TotalSteps != 1234 {
		t.Fatalf("steps: %#v", d.TotalSteps)
	}
	if d.BodyBatteryHighestValue == nil || *d.BodyBatteryHighestValue != 92 {
		t.Fatalf("bodyBatteryHighestValue: %#v", d.BodyBatteryHighestValue)
	}
}
