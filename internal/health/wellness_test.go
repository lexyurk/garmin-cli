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

func wellnessTestClient(t *testing.T, handler http.HandlerFunc) *client.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}
	return client.NewWithSession("ignored", "default", sess, client.Options{HTTPClient: srv.Client(), BaseURL: srv.URL})
}

func TestGetSpO2(t *testing.T) {
	c := wellnessTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wellness-service/wellness/daily/spo2/2026-05-28" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"calendarDate":"2026-05-28","averageSpO2":96,"lowestSpO2":92,"latestSpO2":95}`))
	})
	s, err := GetSpO2(context.Background(), c, "2026-05-28")
	if err != nil {
		t.Fatalf("GetSpO2: %v", err)
	}
	if s.Date != "2026-05-28" || s.Average == nil || *s.Average != 96 || s.Lowest == nil || *s.Lowest != 92 {
		t.Fatalf("unexpected: %#v", s)
	}
}

func TestGetRespiration(t *testing.T) {
	c := wellnessTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wellness-service/wellness/daily/respiration/2026-05-28" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"calendarDate":"2026-05-28","avgWakingRespirationValue":14.5,"highestRespirationValue":22,"lowestRespirationValue":10}`))
	})
	s, err := GetRespiration(context.Background(), c, "2026-05-28")
	if err != nil {
		t.Fatalf("GetRespiration: %v", err)
	}
	if s.AvgWaking == nil || *s.AvgWaking != 14.5 {
		t.Fatalf("avg waking: %#v", s.AvgWaking)
	}
}

func TestGetIntensityMinutes(t *testing.T) {
	c := wellnessTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wellness-service/wellness/daily/im/2026-05-28" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"calendarDate":"2026-05-28","moderateValue":20,"vigorousValue":35,"weeklyGoal":150}`))
	})
	s, err := GetIntensityMinutes(context.Background(), c, "2026-05-28")
	if err != nil {
		t.Fatalf("GetIntensityMinutes: %v", err)
	}
	if s.Moderate == nil || *s.Moderate != 20 || s.Vigorous == nil || *s.Vigorous != 35 {
		t.Fatalf("unexpected: %#v", s)
	}
}

func TestOrDateFallback(t *testing.T) {
	if got := orDate("", "fallback"); got != "fallback" {
		t.Fatalf("expected fallback, got %q", got)
	}
	if got := orDate("2026-01-01", "fallback"); got != "2026-01-01" {
		t.Fatalf("expected calendar date, got %q", got)
	}
}

func TestGetStressDetail(t *testing.T) {
	c := wellnessTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/wellness-service/wellness/dailyStress/2026-07-01" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"calendarDate": "2026-07-01",
			"avgStressLevel": 31,
			"maxStressLevel": 96,
			"stressValuesArray": [[1782900000000, 25], [1782900180000, -2], [1782900360000, 40]]
		}`))
	})
	got, err := GetStressDetail(context.Background(), c, "2026-07-01")
	if err != nil {
		t.Fatalf("GetStressDetail: %v", err)
	}
	if got.Date != "2026-07-01" || got.Average == nil || *got.Average != 31 || got.Max == nil || *got.Max != 96 {
		t.Fatalf("unexpected summary: %+v", got)
	}
	if len(got.Values) != 3 || got.Values[0].Level != 25 || got.Values[1].Level != -2 || got.Values[2].TimestampMs != 1782900360000 {
		t.Fatalf("unexpected values: %+v", got.Values)
	}
}
