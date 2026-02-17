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

func authedClientForTest(srv *httptest.Server) *client.Client {
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}
	return client.NewWithSession("ignored", "default", sess, client.Options{
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
	})
}

func TestSummarizeStatus_PicksFirstKeySortedAndHasFallback(t *testing.T) {
	raw := map[string]any{
		"mostRecentTrainingStatus": map[string]any{
			"payload": map[string]any{
				"latestTrainingStatusData": map[string]any{
					"b": map[string]any{"trainingStatusFeedbackPhrase": "Ignored", "trainingStatus": float64(2)},
					"a": map[string]any{"trainingStatusFeedbackPhrase": "Productive", "trainingStatus": float64(1), "weeklyTrainingLoad": float64(123)},
				},
			},
		},
	}

	s := summarizeStatus("2026-02-16", raw)
	if s.StatusPhrase != "Productive" {
		t.Fatalf("unexpected phrase: %q", s.StatusPhrase)
	}
	if s.StatusID == nil || *s.StatusID != 1 {
		t.Fatalf("unexpected status id: %#v", s.StatusID)
	}
	if s.WeeklyTrainingLoad == nil || *s.WeeklyTrainingLoad != 123 {
		t.Fatalf("unexpected weekly load: %#v", s.WeeklyTrainingLoad)
	}

	empty := summarizeStatus("2026-02-16", map[string]any{})
	if empty.StatusPhrase != "—" {
		t.Fatalf("expected fallback phrase, got %q", empty.StatusPhrase)
	}
}

func TestGetStatus(t *testing.T) {
	date := "2026-02-16"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/mobile-gateway/usersummary/trainingstatus/latest/"+date {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "mostRecentTrainingStatus": {
		    "payload": {
		      "latestTrainingStatusData": {
		        "2026-02-16": {
		          "trainingStatusFeedbackPhrase": "Peaking",
		          "trainingStatus": 3,
		          "weeklyTrainingLoad": 321,
		          "loadLevelTrend": "INCREASING"
		        }
		      }
		    }
		  }
		}`))
	}))
	defer srv.Close()

	c := authedClientForTest(srv)
	s, err := GetStatus(context.Background(), c, date)
	if err != nil {
		t.Fatalf("GetStatus error: %v", err)
	}
	if s.StatusPhrase != "Peaking" || s.LoadLevelTrend != "INCREASING" {
		t.Fatalf("unexpected summary: %#v", s)
	}
	if s.StatusID == nil || *s.StatusID != 3 {
		t.Fatalf("unexpected status id: %#v", s.StatusID)
	}
}

func TestGetVO2Max(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/userprofile-service/userprofile/user-settings" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"userData":{"vo2MaxRunning":50,"vo2MaxCycling":48}}`))
	}))
	defer srv.Close()

	c := authedClientForTest(srv)
	v, err := GetVO2Max(context.Background(), c)
	if err != nil {
		t.Fatalf("GetVO2Max error: %v", err)
	}
	if v.Running != 50 || v.Cycling != 48 {
		t.Fatalf("unexpected vo2max: %#v", v)
	}
}

func TestGetReadiness(t *testing.T) {
	date := "2026-02-16"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metrics-service/metrics/trainingreadiness/"+date {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
		  {"calendarDate":"2026-02-16","timestamp":"2026-02-16T07:00:00.0","level":"LOW","score":30},
		  {"calendarDate":"2026-02-16","timestamp":"2026-02-16T09:00:00.0","level":"HIGH","score":80}
		]`))
	}))
	defer srv.Close()

	c := authedClientForTest(srv)
	s, err := GetReadiness(context.Background(), c, date)
	if err != nil {
		t.Fatalf("GetReadiness error: %v", err)
	}
	if s.Level != "HIGH" || s.Score == nil || *s.Score != 80 {
		t.Fatalf("unexpected readiness: %#v", s)
	}
}

func TestGetHRV(t *testing.T) {
	date := "2026-02-16"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hrv-service/hrv/"+date {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "hrvSummary": {
		    "calendarDate": "",
		    "weeklyAvg": 40,
		    "lastNightAvg": 42,
		    "status": "BALANCED",
		    "baseline": {"lowUpper": 36, "balancedUpper": 52}
		  }
		}`))
	}))
	defer srv.Close()

	c := authedClientForTest(srv)
	s, err := GetHRV(context.Background(), c, date)
	if err != nil {
		t.Fatalf("GetHRV error: %v", err)
	}
	if s.Date != date {
		t.Fatalf("expected fallback date %q, got %q", date, s.Date)
	}
	if s.BaselineLowUpper == nil || *s.BaselineLowUpper != 36 {
		t.Fatalf("unexpected baseline: %#v", s)
	}
}
