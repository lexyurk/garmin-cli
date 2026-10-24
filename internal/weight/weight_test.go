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

func TestLatest_UnsortedMultiDayFixture(t *testing.T) {
	// Mirrors the 2026-10-24 repro: dateRange is not ordered, and the oldest
	// day (2026-09-27) is last. The newest day has three times; the latest
	// instant is neither the first nor the last entry for that day, and it
	// does not have the highest samplePk.
	ms := func(s string) int64 {
		t.Helper()
		parsed, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatal(err)
		}
		return parsed.UnixMilli()
	}
	payload, err := json.Marshal(map[string]any{
		"dateWeightList": []map[string]any{
			{"samplePk": 11, "calendarDate": "2026-10-18", "timestampGMT": ms("2026-10-18T08:00:00Z"), "weight": 71100},
			{"samplePk": 300, "calendarDate": "2026-10-20", "timestampGMT": ms("2026-10-20T07:00:00Z"), "weight": 71000, "bmi": 22.0, "bodyFat": 15.0, "bodyWater": 50.0},
			{"samplePk": 200, "calendarDate": "2026-10-20", "timestampGMT": ms("2026-10-20T21:15:00Z"), "weight": 72739, "bmi": 23.4, "bodyFat": 18.1, "bodyWater": 54.2},
			{"samplePk": 400, "calendarDate": "2026-10-20", "timestampGMT": ms("2026-10-20T12:00:00Z"), "weight": 71500, "bmi": 22.5, "bodyFat": 16.0, "bodyWater": 52.0},
			{"samplePk": 1, "calendarDate": "2026-09-27", "timestampGMT": ms("2026-09-27T09:00:00Z"), "weight": 69500, "bmi": 21.1, "bodyFat": 14.0, "bodyWater": 53.0},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var gotPath, startDate, endDate string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		startDate = r.URL.Query().Get("startDate")
		endDate = r.URL.Query().Get("endDate")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(payload)
	})

	got, err := Latest(context.Background(), c, 30)
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if gotPath != "/weight-service/weight/dateRange" {
		t.Fatalf("path: %s", gotPath)
	}
	now := time.Now().In(time.Local)
	wantEnd := now.Format("2006-01-02")
	wantStart := now.AddDate(0, 0, -29).Format("2006-01-02")
	if startDate != wantStart || endDate != wantEnd {
		t.Fatalf("window start=%q end=%q, want %q..%q", startDate, endDate, wantStart, wantEnd)
	}
	if got.Date != "2026-10-20" || got.SamplePk != 200 {
		t.Fatalf("expected newest weigh-in, got %#v", got)
	}
	wantKG := 72739.0 / 1000.0
	if got.WeightKG == nil || *got.WeightKG != wantKG {
		t.Fatalf("weight_kg: %#v", got.WeightKG)
	}
	if got.BMI == nil || *got.BMI != 23.4 || got.BodyFatPct == nil || *got.BodyFatPct != 18.1 || got.BodyWater == nil || *got.BodyWater != 54.2 {
		t.Fatalf("body composition: %#v", got)
	}

	blob, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(blob, &obj); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"date", "weight_kg", "bmi", "body_fat_pct", "body_water_pct", "sample_pk"} {
		if _, ok := obj[key]; !ok {
			t.Fatalf("missing %s in %s", key, blob)
		}
	}
	if _, ok := obj["timestampGMT"]; ok {
		t.Fatalf("unexpected timestamp field in output: %s", blob)
	}
	if len(obj) != 6 {
		t.Fatalf("output fields changed: %s", blob)
	}
}

func TestLatest_SameDayStringAndDateTimestamps(t *testing.T) {
	// Same calendar day, unsorted. Later timestampGMT string wins over an
	// earlier numeric `date` and over a higher samplePk.
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// date "1792483200000" is 2026-10-20T08:00:00Z, between the timestampGMT strings.
		_, _ = w.Write([]byte(`{"dateWeightList":[
		  {"samplePk":9,"calendarDate":"2026-10-20","date":"1792483200000","weight":70000},
		  {"samplePk":8,"calendarDate":"2026-10-19","timestampGMT":"2026-10-19T08:00:00Z","weight":68000},
		  {"samplePk":7,"calendarDate":"2026-10-20","timestampGMT":"2026-10-20T06:00:00","weight":69000},
		  {"samplePk":1,"calendarDate":"2026-10-20","timestampGMT":"2026-10-20T21:15:00.000","weight":72739},
		  {"samplePk":50,"calendarDate":"2026-10-20","timestampGMT":"2026-10-20T07:00:00.0","weight":71000},
		  {"samplePk":4,"calendarDate":"2026-10-18","timestampGMT":0,"date":"","weight":67000}
		]}`))
	})
	got, err := Latest(context.Background(), c, 30)
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if got.Date != "2026-10-20" || got.SamplePk != 1 {
		t.Fatalf("expected latest timestamp on the day, got %#v", got)
	}
	wantKG := 72739.0 / 1000.0
	if got.WeightKG == nil || *got.WeightKG != wantKG {
		t.Fatalf("weight_kg: %#v", got.WeightKG)
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
