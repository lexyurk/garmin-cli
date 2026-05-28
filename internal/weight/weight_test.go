package weight

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

func TestList_ConvertsGramsToKG(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/weight-service/weight/dateRange" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("startDate") != "2026-05-01" {
			t.Fatalf("startDate: %q", r.URL.Query().Get("startDate"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"dateWeightList":[
		  {"samplePk":9,"calendarDate":"2026-05-02","weight":75000,"bmi":22.5,"bodyFat":15.2}
		]}`))
	})

	out, err := List(context.Background(), c, "2026-05-01", "2026-05-28")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1, got %d", len(out))
	}
	if out[0].WeightKG == nil || *out[0].WeightKG != 75.0 {
		t.Fatalf("grams->kg conversion failed: %#v", out[0].WeightKG)
	}
	if out[0].SamplePk != 9 {
		t.Fatalf("samplePk: %d", out[0].SamplePk)
	}
}

func TestAdd_PostsKgPayload(t *testing.T) {
	var path string
	var body map[string]any
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusNoContent)
	})

	if err := Add(context.Background(), c, 74.5, "2026-05-28"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if path != "/weight-service/user-weight" {
		t.Fatalf("path: %s", path)
	}
	if body["unitKey"] != "kg" {
		t.Fatalf("unitKey: %#v", body["unitKey"])
	}
	if body["value"].(float64) != 74.5 {
		t.Fatalf("value should be kg, not grams: %#v", body["value"])
	}
	if body["sourceType"] != "MANUAL" {
		t.Fatalf("sourceType: %#v", body["sourceType"])
	}
	ts, _ := body["dateTimestamp"].(string)
	if len(ts) != len("2006-01-02T15:04:05.000") || ts[10] != 'T' {
		t.Fatalf("dateTimestamp format: %q", ts)
	}
}

func TestAdd_Validates(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not call API")
	})
	if err := Add(context.Background(), c, 0, ""); err == nil {
		t.Fatalf("expected error for non-positive weight")
	}
	if err := Add(context.Background(), c, 70, "not-a-date"); err == nil {
		t.Fatalf("expected error for bad date")
	}
}

func TestLatest_ReturnsMostRecentAndErrors(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"dateWeightList":[
		  {"calendarDate":"2026-05-01","weight":76000},
		  {"calendarDate":"2026-05-28","weight":75000}
		]}`))
	})
	w, err := Latest(context.Background(), c, 30)
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if w.Date != "2026-05-28" || w.WeightKG == nil || *w.WeightKG != 75 {
		t.Fatalf("unexpected latest: %#v", w)
	}

	empty := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"dateWeightList":[]}`))
	})
	if _, err := Latest(context.Background(), empty, 0); err == nil {
		t.Fatalf("expected error when no weigh-ins")
	}
}

func TestList_RejectsBadDates(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) { t.Fatalf("should not call API") })
	if _, err := List(context.Background(), c, "bad", "2026-05-28"); err == nil {
		t.Fatalf("expected error for bad start date")
	}
	if _, err := List(context.Background(), c, "2026-05-01", "nope"); err == nil {
		t.Fatalf("expected error for bad end date")
	}
}
