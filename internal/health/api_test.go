package health

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lexyurk/garmin-cli/internal/auth"
	"github.com/lexyurk/garmin-cli/internal/client"
)

func TestGetDailySummary_AndCalendarDateOr(t *testing.T) {
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/usersummary-service/usersummary/daily/" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ok" {
			t.Fatalf("unexpected Authorization: %q", got)
		}
		date := r.URL.Query().Get("calendarDate")
		w.Header().Set("Content-Type", "application/json")
		// Intentionally leave calendarDate empty to exercise CalendarDateOr fallback.
		_, _ = w.Write([]byte(`{"calendarDate":"","totalSteps":123,"restingHeartRate":55,"stressQualifier":"LOW"}`))
		if date == "" {
			t.Fatalf("expected calendarDate query")
		}
	}))
	defer srv.Close()

	c := client.NewWithSession("ignored", "default", sess, client.Options{HTTPClient: srv.Client(), BaseURL: srv.URL})

	d, err := GetDailySummary(context.Background(), c, "2026-02-16")
	if err != nil {
		t.Fatalf("GetDailySummary error: %v", err)
	}
	if got := d.CalendarDateOr("fallback"); got != "fallback" {
		t.Fatalf("expected fallback date, got %q", got)
	}
	if d.TotalSteps == nil || *d.TotalSteps != 123 {
		t.Fatalf("unexpected steps: %#v", d.TotalSteps)
	}
}

func TestGetSleep_UsesFallbackDate(t *testing.T) {
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sleep-service/sleep/dailySleepData" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("date") == "" {
			t.Fatalf("expected date query param")
		}
		w.Header().Set("Content-Type", "application/json")
		// No calendarDate in payload -> should fall back to request date.
		_, _ = w.Write([]byte(`{"dailySleepDTO":{"calendarDate":"","sleepTimeSeconds":3600,"sleepScores":{"overall":{"value":80}}}}`))
	}))
	defer srv.Close()

	c := client.NewWithSession("ignored", "default", sess, client.Options{HTTPClient: srv.Client(), BaseURL: srv.URL})

	s, err := GetSleep(context.Background(), c, "2026-02-16")
	if err != nil {
		t.Fatalf("GetSleep error: %v", err)
	}
	if s.Date != "2026-02-16" {
		t.Fatalf("expected fallback date, got %q", s.Date)
	}
	if s.Score == nil || *s.Score != 80 {
		t.Fatalf("unexpected score: %#v", s.Score)
	}
}
