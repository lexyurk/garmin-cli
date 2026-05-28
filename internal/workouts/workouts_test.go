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
