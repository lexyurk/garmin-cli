package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lexyurk/garmin-cli/internal/auth"
)

func writeTestSession(t *testing.T, cfgDir string) int64 {
	t.Helper()
	expiresAt := time.Now().Add(1 * time.Hour).Unix()
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: expiresAt},
	}
	if err := auth.SaveSession(cfgDir, "", sess); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}
	return expiresAt
}

func runCLI(t *testing.T, cfgDir string, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	cmd := NewRootCmd("dev")
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs(append([]string{"--config-dir", cfgDir}, args...))

	err = cmd.Execute()
	return out.String(), errOut.String(), err
}

func newConnectAPIServer(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()

	checkAuth := func(r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer ok" {
			t.Fatalf("unexpected Authorization header: %q", got)
		}
	}

	mux.HandleFunc("/userprofile-service/socialProfile", func(w http.ResponseWriter, r *http.Request) {
		checkAuth(r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"displayName":"Test User"}`))
	})

	mux.HandleFunc("/usersummary-service/usersummary/daily/", func(w http.ResponseWriter, r *http.Request) {
		checkAuth(r)
		date := r.URL.Query().Get("calendarDate")
		if date == "" {
			t.Fatalf("missing calendarDate query")
		}
		cal := date
		if date == "2026-02-15" {
			// Exercise CalendarDateOr fallback in CLI.
			cal = ""
		}
		resp := map[string]any{
			"calendarDate":               cal,
			"totalSteps":                 1234,
			"dailyStepGoal":              8000,
			"totalDistanceMeters":        950,
			"minHeartRate":               45,
			"maxHeartRate":               155,
			"restingHeartRate":           52,
			"averageStressLevel":         23,
			"maxStressLevel":             86,
			"stressQualifier":            "LOW",
			"bodyBatteryChargedValue":    50,
			"bodyBatteryDrainedValue":    60,
			"bodyBatteryHighestValue":    92,
			"bodyBatteryLowestValue":     35,
			"bodyBatteryMostRecentValue": 38,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/sleep-service/sleep/dailySleepData", func(w http.ResponseWriter, r *http.Request) {
		checkAuth(r)
		date := r.URL.Query().Get("date")
		if date == "" {
			t.Fatalf("missing date query")
		}
		cal := date
		if date == "2026-02-15" {
			cal = ""
		}
		resp := map[string]any{
			"dailySleepDTO": map[string]any{
				"calendarDate":            cal,
				"sleepTimeSeconds":        3600,
				"deepSleepSeconds":        900,
				"lightSleepSeconds":       2000,
				"remSleepSeconds":         600,
				"awakeSleepSeconds":       100,
				"averageSpO2Value":        94.0,
				"averageRespirationValue": 13.5,
				"sleepScores": map[string]any{
					"overall": map[string]any{"value": 80},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})

	// Training status: prefix match for date.
	mux.HandleFunc("/mobile-gateway/usersummary/trainingstatus/latest/", func(w http.ResponseWriter, r *http.Request) {
		checkAuth(r)
		date := strings.TrimPrefix(r.URL.Path, "/mobile-gateway/usersummary/trainingstatus/latest/")
		if date == "" {
			t.Fatalf("missing date suffix")
		}
		raw := map[string]any{
			"mostRecentTrainingStatus": map[string]any{
				"payload": map[string]any{
					"latestTrainingStatusData": map[string]any{
						date: map[string]any{
							"trainingStatusFeedbackPhrase": "Productive",
							"trainingStatus":               1,
							"weeklyTrainingLoad":           123,
							"loadLevelTrend":               "STABLE",
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(raw)
	})

	mux.HandleFunc("/metrics-service/metrics/trainingreadiness/", func(w http.ResponseWriter, r *http.Request) {
		checkAuth(r)
		date := strings.TrimPrefix(r.URL.Path, "/metrics-service/metrics/trainingreadiness/")
		out := []map[string]any{
			{"calendarDate": date, "timestamp": date + "T07:00:00.0", "level": "LOW", "score": 30},
			{"calendarDate": date, "timestamp": date + "T09:00:00.0", "level": "HIGH", "score": 80},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})

	mux.HandleFunc("/hrv-service/hrv/", func(w http.ResponseWriter, r *http.Request) {
		checkAuth(r)
		date := strings.TrimPrefix(r.URL.Path, "/hrv-service/hrv/")
		out := map[string]any{
			"hrvSummary": map[string]any{
				"calendarDate": "",
				"weeklyAvg":    40,
				"lastNightAvg": 42,
				"status":       "BALANCED",
				"baseline":     map[string]any{"lowUpper": 36.0, "balancedUpper": 52.0},
			},
		}
		// Ensure we exercise the fallback date logic in ToSummary (calendarDate empty).
		_ = date
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})

	mux.HandleFunc("/userprofile-service/userprofile/user-settings", func(w http.ResponseWriter, r *http.Request) {
		checkAuth(r)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"userData":{"vo2MaxRunning":50,"vo2MaxCycling":48}}`))
	})

	mux.HandleFunc("/activitylist-service/activities/search/activities", func(w http.ResponseWriter, r *http.Request) {
		checkAuth(r)
		start := r.URL.Query().Get("start")
		limit := r.URL.Query().Get("limit")
		if start != "0" {
			_ = json.NewEncoder(w).Encode([]any{})
			return
		}
		if limit == "" {
			t.Fatalf("missing limit query")
		}
		// Always return 2 activities; CLI will request --limit 2 in tests.
		out := []map[string]any{
			{
				"activityId":     123,
				"activityName":   "Morning Run",
				"startTimeLocal": "2026-02-16 07:00:00",
				"activityType":   map[string]any{"typeKey": "running"},
				"distance":       5000,
				"duration":       1500,
				"calories":       321,
				"averageHR":      140,
			},
			{
				"activityId":     124,
				"activityName":   "Easy Ride",
				"startTimeLocal": "2026-02-16 08:00:00",
				"activityType":   map[string]any{"typeKey": "cycling"},
				"distance":       20000,
				"duration":       3600,
				"calories":       500,
				"averageHR":      130,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})

	mux.HandleFunc("/activity-service/activity/", func(w http.ResponseWriter, r *http.Request) {
		checkAuth(r)
		id := strings.TrimPrefix(r.URL.Path, "/activity-service/activity/")
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{
		  "activityName":"Morning Run",
		  "startTimeLocal":"2026-02-16 07:00:00",
		  "activityType":{"typeKey":"running"},
		  "distance":5000,
		  "duration":1500,
		  "calories":321,
		  "averageHR":140,
		  "maxHR":180,
		  "splitSummaries":[
		    {"distance":1000,"duration":300,"averageHR":150,"maxHR":170},
		    {"distance":1000,"duration":290}
		  ]
		}`)
		_ = id
	})

	mux.HandleFunc("/download-service/export/gpx/activity/", func(w http.ResponseWriter, r *http.Request) {
		checkAuth(r)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("file-contents"))
	})

	return httptest.NewServer(mux)
}

func TestCLI_AuthenticatedFlows(t *testing.T) {
	srv := newConnectAPIServer(t)
	defer srv.Close()
	t.Setenv("GARMIN_CONNECTAPI_BASE_URL", srv.URL)

	cfgDir := t.TempDir()
	expiresAt := writeTestSession(t, cfgDir)

	t.Run("auth status json", func(t *testing.T) {
		stdout, stderr, err := runCLI(t, cfgDir, "--format", "json", "auth", "status")
		if err != nil {
			t.Fatalf("unexpected err: %v\nstderr:\n%s", err, stderr)
		}
		if strings.TrimSpace(stderr) != "" {
			t.Fatalf("expected empty stderr, got:\n%s", stderr)
		}
		if !strings.Contains(stdout, `"authenticated": true`) || !strings.Contains(stdout, `"display_name": "Test User"`) {
			t.Fatalf("unexpected stdout:\n%s", stdout)
		}
		if !strings.Contains(stdout, fmt.Sprintf(`"expires_at": %d`, expiresAt)) {
			t.Fatalf("expected expires_at in stdout, got:\n%s", stdout)
		}
	})

	t.Run("health summary markdown single-date", func(t *testing.T) {
		stdout, _, err := runCLI(t, cfgDir, "health", "summary", "--date", "2026-02-16")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(stdout, "## Daily summary") || !strings.Contains(stdout, "steps") {
			t.Fatalf("unexpected output:\n%s", stdout)
		}
	})

	t.Run("health summary table multi-date", func(t *testing.T) {
		stdout, _, err := runCLI(t, cfgDir, "--format", "table", "health", "summary", "--from", "2026-02-15", "--to", "2026-02-16")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(stdout, "date") || !strings.Contains(stdout, "steps") {
			t.Fatalf("unexpected output:\n%s", stdout)
		}
	})

	t.Run("health sleep markdown", func(t *testing.T) {
		stdout, _, err := runCLI(t, cfgDir, "health", "sleep", "--date", "2026-02-16")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(stdout, "## Sleep") || !strings.Contains(stdout, "total") {
			t.Fatalf("unexpected output:\n%s", stdout)
		}
	})

	t.Run("health heart-rate json", func(t *testing.T) {
		stdout, _, err := runCLI(t, cfgDir, "--format", "json", "health", "heart-rate", "--date", "2026-02-16")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(stdout, `"resting_hr"`) {
			t.Fatalf("unexpected output:\n%s", stdout)
		}
	})

	t.Run("health steps markdown", func(t *testing.T) {
		stdout, _, err := runCLI(t, cfgDir, "health", "steps", "--date", "2026-02-16")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(stdout, "## Steps") || !strings.Contains(stdout, "distance") {
			t.Fatalf("unexpected output:\n%s", stdout)
		}
	})

	t.Run("health stress table", func(t *testing.T) {
		stdout, _, err := runCLI(t, cfgDir, "--format", "table", "health", "stress", "--from", "2026-02-15", "--to", "2026-02-16")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(stdout, "qualifier") || !strings.Contains(stdout, "LOW") {
			t.Fatalf("unexpected output:\n%s", stdout)
		}
	})

	t.Run("health body-battery markdown", func(t *testing.T) {
		stdout, _, err := runCLI(t, cfgDir, "health", "body-battery", "--date", "2026-02-16")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(stdout, "## Body battery") || !strings.Contains(stdout, "highest") {
			t.Fatalf("unexpected output:\n%s", stdout)
		}
	})

	t.Run("training status markdown", func(t *testing.T) {
		stdout, _, err := runCLI(t, cfgDir, "training", "status", "--date", "2026-02-16")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(stdout, "## Training status") || !strings.Contains(stdout, "Productive") {
			t.Fatalf("unexpected output:\n%s", stdout)
		}
	})

	t.Run("training readiness json", func(t *testing.T) {
		stdout, _, err := runCLI(t, cfgDir, "--format", "json", "training", "readiness", "--date", "2026-02-16")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(stdout, `"score": 80`) {
			t.Fatalf("unexpected output:\n%s", stdout)
		}
	})

	t.Run("training vo2max markdown", func(t *testing.T) {
		stdout, _, err := runCLI(t, cfgDir, "training", "vo2max")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(stdout, "## VO2 max") || !strings.Contains(stdout, "running") {
			t.Fatalf("unexpected output:\n%s", stdout)
		}
	})

	t.Run("training hrv table", func(t *testing.T) {
		stdout, _, err := runCLI(t, cfgDir, "--format", "table", "training", "hrv", "--from", "2026-02-15", "--to", "2026-02-16")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(stdout, "weekly_avg") || !strings.Contains(stdout, "BALANCED") {
			t.Fatalf("unexpected output:\n%s", stdout)
		}
	})

	t.Run("activities list markdown", func(t *testing.T) {
		stdout, _, err := runCLI(t, cfgDir, "activities", "list", "--date", "2026-02-16", "--limit", "2")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(stdout, "| id |") || !strings.Contains(stdout, "Morning Run") {
			t.Fatalf("unexpected output:\n%s", stdout)
		}
	})

	t.Run("activities get markdown", func(t *testing.T) {
		stdout, _, err := runCLI(t, cfgDir, "activities", "get", "123")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(stdout, "## Activity") || !strings.Contains(stdout, "Morning Run") {
			t.Fatalf("unexpected output:\n%s", stdout)
		}
	})

	t.Run("activities splits json", func(t *testing.T) {
		stdout, _, err := runCLI(t, cfgDir, "--format", "json", "activities", "splits", "123")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(stdout, `"activity_id": 123`) || !strings.Contains(stdout, `"splits"`) {
			t.Fatalf("unexpected output:\n%s", stdout)
		}
	})

	t.Run("activities splits markdown (pace)", func(t *testing.T) {
		stdout, _, err := runCLI(t, cfgDir, "activities", "splits", "123")
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if !strings.Contains(stdout, "pace_min_per_km") || !strings.Contains(stdout, "5:00") {
			t.Fatalf("unexpected output:\n%s", stdout)
		}
	})

	t.Run("activities export to file", func(t *testing.T) {
		outPath := filepath.Join(t.TempDir(), "activity.gpx")
		stdout, stderr, err := runCLI(t, cfgDir, "activities", "export", "123", "--type", "gpx", "--out", outPath)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if strings.TrimSpace(stdout) != "" {
			t.Fatalf("expected clean stdout when writing to file, got:\n%s", stdout)
		}
		if !strings.Contains(stderr, "downloaded") {
			t.Fatalf("expected download confirmation on stderr, got:\n%s", stderr)
		}
		b, err := os.ReadFile(outPath)
		if err != nil {
			t.Fatalf("read outPath: %v", err)
		}
		if string(b) != "file-contents" {
			t.Fatalf("unexpected file contents: %q", string(b))
		}
	})

	t.Run("auth logout json", func(t *testing.T) {
		// First, exercise the human/markdown rendering branch.
		stdout, stderr, err := runCLI(t, cfgDir, "auth", "logout")
		if err != nil {
			t.Fatalf("unexpected err: %v\nstderr:\n%s", err, stderr)
		}
		if strings.TrimSpace(stderr) != "" {
			t.Fatalf("expected empty stderr, got:\n%s", stderr)
		}
		if !strings.Contains(stdout, "## Logged out") {
			t.Fatalf("unexpected output:\n%s", stdout)
		}

		// Then, exercise JSON output too.
		stdout, stderr, err = runCLI(t, cfgDir, "--format", "json", "auth", "logout")
		if err != nil {
			t.Fatalf("unexpected err: %v\nstderr:\n%s", err, stderr)
		}
		if strings.TrimSpace(stderr) != "" {
			t.Fatalf("expected empty stderr, got:\n%s", stderr)
		}
		if !strings.Contains(stdout, `"ok": true`) {
			t.Fatalf("unexpected output:\n%s", stdout)
		}
		if _, err := os.Stat(filepath.Join(cfgDir, "tokens", "default", "oauth1_token.json")); !os.IsNotExist(err) {
			t.Fatalf("expected tokens to be removed, stat err=%v", err)
		}
	})
}
