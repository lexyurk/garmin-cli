package training

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lexyurk/garmin-cli/internal/auth"
	"github.com/lexyurk/garmin-cli/internal/client"
)

func raceTestClient(t *testing.T, handler http.HandlerFunc) *client.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}
	return client.NewWithSession("ignored", "default", sess, client.Options{HTTPClient: srv.Client(), BaseURL: srv.URL})
}

func TestGetRacePredictions(t *testing.T) {
	c := raceTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics-service/metrics/racepredictions/latest/runner" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"calendarDate":"2026-05-28","time5K":1350,"time10K":2820,"timeHalfMarathon":6300,"timeMarathon":13500}`))
	})

	rp, err := GetRacePredictions(context.Background(), c, "runner")
	if err != nil {
		t.Fatalf("GetRacePredictions: %v", err)
	}
	if rp.Time5KSeconds == nil || *rp.Time5KSeconds != 1350 {
		t.Fatalf("5k: %#v", rp.Time5KSeconds)
	}
	if rp.TimeMarathonSeconds == nil || *rp.TimeMarathonSeconds != 13500 {
		t.Fatalf("marathon: %#v", rp.TimeMarathonSeconds)
	}
}

func TestGetRacePredictions_RequiresDisplayName(t *testing.T) {
	c := raceTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not call API")
	})
	if _, err := GetRacePredictions(context.Background(), c, "  "); err == nil {
		t.Fatalf("expected error for empty display name")
	}
}
