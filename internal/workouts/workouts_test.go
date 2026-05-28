package workouts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestList_ParsesWorkouts(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/workout-service/workouts" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("limit"); got != "5" {
			t.Fatalf("limit: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
		  {"workoutId":111,"workoutName":"Intervals","sportType":{"sportTypeId":1,"sportTypeKey":"running"},"estimatedDurationInSecs":3600,"estimatedDistanceInMeters":10000}
		]`))
	})

	out, err := List(context.Background(), c, 5)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(out) != 1 || out[0].WorkoutID != 111 || out[0].Sport != "running" {
		t.Fatalf("unexpected: %#v", out)
	}
	if out[0].DurationSecs != 3600 {
		t.Fatalf("duration: %v", out[0].DurationSecs)
	}
}

func TestSummarizeAndCountSteps(t *testing.T) {
	raw := map[string]any{}
	body := `{
	  "workoutName": "Track day",
	  "description": "speed",
	  "sportType": {"sportTypeId":1,"sportTypeKey":"running"},
	  "estimatedDurationInSecs": 2400,
	  "workoutSegments": [
	    {"workoutSteps": [
	      {"stepType":{"stepTypeKey":"warmup"}},
	      {"numberOfIterations": 4, "workoutSteps": [
	        {"stepType":{"stepTypeKey":"interval"}},
	        {"stepType":{"stepTypeKey":"recovery"}}
	      ]},
	      {"stepType":{"stepTypeKey":"cooldown"}}
	    ]}
	  ]
	}`
	if err := json.Unmarshal([]byte(body), &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	s := SummarizeRaw(111, raw)
	if s.Name != "Track day" || s.Sport != "running" || s.DurationSecs != 2400 {
		t.Fatalf("summary: %#v", s)
	}
	// warmup + 4*(interval+recovery) + cooldown = 1 + 8 + 1 = 10
	if got := CountSteps(raw); got != 10 {
		t.Fatalf("CountSteps: %d", got)
	}
}

func TestDelete_SendsDelete(t *testing.T) {
	var method, path string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	if err := Delete(context.Background(), c, 111); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if method != http.MethodDelete || path != "/workout-service/workout/111" {
		t.Fatalf("unexpected request: %s %s", method, path)
	}
}

func TestUpdate_FetchesAndPutsName(t *testing.T) {
	var putBody map[string]any
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"workoutName":"old","description":"d","sportType":{"sportTypeKey":"running"}}`))
		case http.MethodPut:
			_ = json.NewDecoder(r.Body).Decode(&putBody)
			w.WriteHeader(http.StatusNoContent)
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	})

	name := "new name"
	s, err := Update(context.Background(), c, 111, UpdateOptions{Name: &name})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if putBody["workoutName"] != "new name" {
		t.Fatalf("workoutName not set in PUT body: %#v", putBody)
	}
	// description preserved from the fetched object
	if putBody["description"] != "d" {
		t.Fatalf("description not preserved: %#v", putBody)
	}
	if s.Name != "new name" {
		t.Fatalf("summary name: %q", s.Name)
	}
}

func TestUpdate_NothingToUpdate(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not call API")
	})
	if _, err := Update(context.Background(), c, 111, UpdateOptions{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestSchedule_PostsDate(t *testing.T) {
	var path string
	var body map[string]any
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"workoutScheduleId":9001}`))
	})

	res, err := Schedule(context.Background(), c, 111, "2026-06-01")
	if err != nil {
		t.Fatalf("Schedule: %v", err)
	}
	if path != "/workout-service/schedule/111" {
		t.Fatalf("path: %s", path)
	}
	if body["date"] != "2026-06-01" {
		t.Fatalf("date not in body: %#v", body)
	}
	if res.WorkoutScheduleID != 9001 || res.Date != "2026-06-01" {
		t.Fatalf("unexpected result: %#v", res)
	}
}

func TestSchedule_RejectsBadDate(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not call API")
	})
	if _, err := Schedule(context.Background(), c, 111, "06/01/2026"); err == nil {
		t.Fatalf("expected error for bad date")
	}
}
