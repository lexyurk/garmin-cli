package calendar

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lexyurk/garmin-cli/internal/auth"
	"github.com/lexyurk/garmin-cli/internal/client"
)

func testClient(t *testing.T, handler http.HandlerFunc) *client.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}
	return client.NewWithSession("ignored", "default", sess, client.Options{HTTPClient: srv.Client(), BaseURL: srv.URL})
}

func TestMonth_UsesZeroBasedMonth(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		// June (6) -> 0-based 5
		if r.URL.Path != "/calendar-service/year/2026/month/5" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"calendarItems":[
		  {"date":"2026-06-03","itemType":"workout","title":"Intervals","workoutId":111},
		  {"calendarDate":"2026-06-05","itemType":"activity","title":"Run","activityId":222}
		]}`))
	})

	items, err := Month(context.Background(), c, 2026, 6)
	if err != nil {
		t.Fatalf("Month: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].WorkoutID != 111 || items[0].Date != "2026-06-03" || items[0].Title != "Intervals" || items[0].Type != "workout" {
		t.Fatalf("item0: %#v", items[0])
	}
	if items[0].ScheduleID != 0 {
		t.Fatalf("missing id copied into schedule_id: %#v", items[0])
	}
	if items[1].Date != "2026-06-05" || items[1].ActivityID != 222 || items[1].Title != "Run" || items[1].Type != "activity" {
		t.Fatalf("item1: %#v", items[1])
	}
	if items[1].ScheduleID != 0 || items[1].WorkoutID != 0 {
		t.Fatalf("item1 ids: %#v", items[1])
	}
}

func TestMonth_RejectsBadMonth(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not call API")
	})
	if _, err := Month(context.Background(), c, 2026, 13); err == nil {
		t.Fatalf("expected error for month 13")
	}
}

func monthFrom(t *testing.T, body string) ([]Item, error) {
	t.Helper()
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/calendar-service/year/2026/month/9" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	})
	return Month(context.Background(), c, 2026, 10)
}

func TestMonth_ScheduleIDDistinguishesSharedWorkout(t *testing.T) {
	items, err := monthFrom(t, `{"calendarItems":[
	  {"date":"2026-10-20","itemType":"workout","title":"30min Recovery Run","workoutId":111,"id":9001},
	  {"date":"2026-10-22","itemType":"workout","title":"30min Recovery Run","workoutId":111,"id":9002},
	  {"date":"2026-10-24","itemType":"workout","title":"30min Recovery Run","workoutId":111,"id":-4}
	]}`)
	if err != nil {
		t.Fatalf("Month: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[0].WorkoutID != 111 || items[1].WorkoutID != 111 || items[0].ScheduleID != 9001 || items[1].ScheduleID != 9002 {
		t.Fatalf("shared workout rows: %#v", items)
	}
	if items[0].Date != "2026-10-20" || items[1].Date != "2026-10-22" {
		t.Fatalf("dates: %#v", items)
	}
	if items[2].ScheduleID != -4 || items[2].WorkoutID != 111 {
		t.Fatalf("negative id: %#v", items[2])
	}
	raw, err := json.Marshal(items[0])
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(raw), `"schedule_id":9001`) || !strings.Contains(string(raw), `"workout_id":111`) {
		t.Fatalf("json: %s", raw)
	}
}

func TestMonth_ActivityScheduleID(t *testing.T) {
	items, err := monthFrom(t, `{"calendarItems":[
	  {"date":"2026-10-21","itemType":"activity","title":"Run","activityId":222,"id":7001}
	]}`)
	if err != nil {
		t.Fatalf("Month: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	got := items[0]
	if got.ScheduleID != 7001 || got.ActivityID != 222 || got.WorkoutID != 0 {
		t.Fatalf("activity row: %#v", got)
	}
	if got.Date != "2026-10-21" || got.Type != "activity" || got.Title != "Run" {
		t.Fatalf("activity fields: %#v", got)
	}
}

func TestMonth_MissingNullAndZeroID(t *testing.T) {
	items, err := monthFrom(t, `{"calendarItems":[
	  {"date":"2026-10-01","itemType":"workout","title":"Missing","workoutId":111},
	  {"date":"2026-10-02","itemType":"workout","title":"Null","workoutId":222,"id":null},
	  {"date":"2026-10-03","itemType":"workout","title":"Zero","workoutId":333,"id":0}
	]}`)
	if err != nil {
		t.Fatalf("Month: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	wantWorkout := []int64{111, 222, 333}
	for i, it := range items {
		if it.ScheduleID != 0 {
			t.Fatalf("item %d schedule_id = %d, want 0 (%#v)", i, it.ScheduleID, it)
		}
		if it.WorkoutID != wantWorkout[i] {
			t.Fatalf("item %d workout_id = %d", i, it.WorkoutID)
		}
		raw, err := json.Marshal(it)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if strings.Contains(string(raw), "schedule_id") {
			t.Fatalf("item %d json included schedule_id: %s", i, raw)
		}
		if !strings.Contains(string(raw), `"workout_id":`) {
			t.Fatalf("item %d dropped workout_id: %s", i, raw)
		}
	}
}

func TestMonth_IgnoresUnknownKeys(t *testing.T) {
	items, err := monthFrom(t, `{"calendarItems":[
	  {"date":"2026-10-04","itemType":"workout","title":"Intervals","workoutId":111,"activityId":222,"id":9001,"ownerDisplayName":"A & B","mystery":true}
	]}`)
	if err != nil {
		t.Fatalf("Month: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	got := items[0]
	if got.Date != "2026-10-04" || got.Type != "workout" || got.Title != "Intervals" || got.WorkoutID != 111 || got.ActivityID != 222 || got.ScheduleID != 9001 {
		t.Fatalf("item: %#v", got)
	}
}

func TestMonth_StringIDFailsDecode(t *testing.T) {
	_, err := monthFrom(t, `{"calendarItems":[
	  {"date":"2026-10-04","itemType":"workout","title":"Intervals","workoutId":111,"id":"9001"}
	]}`)
	if err == nil {
		t.Fatal("expected string id to fail the month decode")
	}
}

func TestFilterByType(t *testing.T) {
	items := []Item{{Type: "workout", ScheduleID: 9001}, {Type: "activity", ScheduleID: 7001}, {Type: "workout"}}
	filtered := FilterByType(items, "workout")
	if len(filtered) != 2 {
		t.Fatalf("workout: %d", len(filtered))
	}
	if filtered[0].ScheduleID != 9001 {
		t.Fatalf("filtered schedule_id: %#v", filtered[0])
	}
	if got := len(FilterByType(items, "")); got != 3 {
		t.Fatalf("all: %d", got)
	}
}
