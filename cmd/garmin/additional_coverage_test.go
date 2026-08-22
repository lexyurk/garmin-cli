package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runSuccessCases(t *testing.T, cfgDir string, cases []struct {
	name string
	args []string
	want string
}) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := runCLI(t, cfgDir, tc.args...)
			if err != nil {
				t.Fatalf("runCLI: %v\nstderr:\n%s", err, stderr)
			}
			if !strings.Contains(stdout, tc.want) {
				t.Fatalf("expected %q in stdout, got:\n%s", tc.want, stdout)
			}
		})
	}
}

func TestCLI_AdditionalAuthenticatedOutputBranches(t *testing.T) {
	srv := newConnectAPIServer(t)
	defer srv.Close()
	t.Setenv("GARMIN_CONNECTAPI_BASE_URL", srv.URL)

	cfgDir := t.TempDir()
	writeTestSession(t, cfgDir)

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"health summary json", []string{"--format", "json", "health", "summary", "--date", "2026-02-16"}, `"steps": 1234`},
		{"health sleep json", []string{"--format", "json", "health", "sleep", "--date", "2026-02-16"}, `"score": 80`},
		{"health sleep table", []string{"--format", "table", "health", "sleep", "--from", "2026-02-15", "--to", "2026-02-16"}, "sleep_start"},
		{"health heart rate markdown", []string{"health", "heart-rate", "--date", "2026-02-16"}, "## Heart rate"},
		{"health heart rate table", []string{"--format", "table", "health", "heart-rate", "--from", "2026-02-15", "--to", "2026-02-16"}, "resting_hr"},
		{"health steps json", []string{"--format", "json", "health", "steps", "--date", "2026-02-16"}, `"total_steps": 1234`},
		{"health steps table", []string{"--format", "table", "health", "steps", "--from", "2026-02-15", "--to", "2026-02-16"}, "distance_km"},
		{"health stress markdown", []string{"health", "stress", "--date", "2026-02-16"}, "## Stress"},
		{"health stress json", []string{"--format", "json", "health", "stress", "--date", "2026-02-16"}, `"average": 23`},
		{"health body battery json", []string{"--format", "json", "health", "body-battery", "--date", "2026-02-16"}, `"highest": 92`},
		{"health body battery table", []string{"--format", "table", "health", "body-battery", "--from", "2026-02-15", "--to", "2026-02-16"}, "most_recent"},
		{"training status json", []string{"--format", "json", "training", "status", "--date", "2026-02-16"}, `"status_phrase": "Productive"`},
		{"training status table", []string{"--format", "table", "training", "status", "--from", "2026-02-15", "--to", "2026-02-16"}, "weekly_load"},
		{"training readiness markdown", []string{"training", "readiness", "--date", "2026-02-16"}, "## Training readiness"},
		{"training readiness table", []string{"--format", "table", "training", "readiness", "--from", "2026-02-15", "--to", "2026-02-16"}, "acute_load"},
		{"training vo2max json", []string{"--format", "json", "training", "vo2max"}, `"running": 50`},
		{"training hrv json", []string{"--format", "json", "training", "hrv", "--date", "2026-02-16"}, `"status": "BALANCED"`},
		{"training hrv markdown", []string{"training", "hrv", "--date", "2026-02-16"}, "## HRV"},
		{"activities list json", []string{"--format", "json", "activities", "list", "--date", "2026-02-16", "--limit", "2"}, `"id": 123`},
		{"activities get json", []string{"--format", "json", "activities", "get", "123"}, `"activityName": "Morning Run"`},
		{"activities get details", []string{"activities", "get", "123", "--details"}, "training_load"},
	}
	runSuccessCases(t, cfgDir, cases)

	t.Run("activities export to stdout", func(t *testing.T) {
		stdout, stderr, err := runCLI(t, cfgDir, "activities", "export", "123")
		if err != nil {
			t.Fatalf("runCLI: %v\nstderr:\n%s", err, stderr)
		}
		if stdout != "file-contents" {
			t.Fatalf("unexpected export: %q", stdout)
		}
	})
}

func TestCLI_AdditionalFeatureOutputBranches(t *testing.T) {
	srv := newFeatureAPIServer(t)
	defer srv.Close()
	t.Setenv("GARMIN_CONNECTAPI_BASE_URL", srv.URL)

	cfgDir := t.TempDir()
	writeTestSession(t, cfgDir)

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"spo2 json", []string{"--format", "json", "health", "spo2", "--date", "2026-05-28"}, `"average": 96`},
		{"spo2 table", []string{"--format", "table", "health", "spo2", "--from", "2026-05-27", "--to", "2026-05-28"}, "lowest"},
		{"respiration json", []string{"--format", "json", "health", "respiration", "--date", "2026-05-28"}, `"avg_waking": 14.5`},
		{"respiration table", []string{"--format", "table", "health", "respiration", "--from", "2026-05-27", "--to", "2026-05-28"}, "highest"},
		{"intensity json", []string{"--format", "json", "health", "intensity-minutes", "--date", "2026-05-28"}, `"moderate": 20`},
		{"intensity table", []string{"--format", "table", "health", "intensity-minutes", "--from", "2026-05-27", "--to", "2026-05-28"}, "vigorous"},
		{"gear list json", []string{"--format", "json", "gear", "list", "--all"}, `"uuid": "u1"`},
		{"gear get json", []string{"--format", "json", "gear", "get", "Pegasus 40"}, `"name": "Pegasus 40"`},
		{"gear stats json", []string{"--format", "json", "gear", "stats", "Pegasus 40"}, `"total_meters": 123456`},
		{"gear add json", []string{"--format", "json", "gear", "add", "--name", "New Shoe"}, `"uuid": "u2"`},
		{"gear retire json", []string{"--format", "json", "gear", "retire", "Pegasus 40"}, `"status": "retired"`},
		{"gear for activity json", []string{"--format", "json", "gear", "for-activity", "123"}, `"uuid": "u1"`},
		{"gear activities json", []string{"--format", "json", "gear", "activities", "Pegasus 40"}, `"id": 123`},
		{"workouts list json", []string{"--format", "json", "workouts", "list"}, `"workout_id": 111`},
		{"workouts get json", []string{"--format", "json", "workouts", "get", "111"}, `"workoutName": "Intervals"`},
		{"workouts update json", []string{"--format", "json", "workouts", "update", "111", "--description", "new"}, `"workout_id": 111`},
		{"workouts schedule json", []string{"--format", "json", "workouts", "schedule", "111", "--date", "2026-06-01"}, `"workout_schedule_id": 9001`},
		{"training fitness age json", []string{"--format", "json", "training", "fitness-age", "--date", "2026-05-28"}, `"fitness_age": 32.5`},
		{"training race predictions json", []string{"--format", "json", "training", "race-predictions"}, `"time_5k_seconds": 1350`},
		{"weight list json", []string{"--format", "json", "weight", "list", "--days", "30"}, `"weight_kg": 75`},
		{"weight latest json", []string{"--format", "json", "weight", "latest"}, `"sample_pk": 9`},
		{"devices json", []string{"--format", "json", "devices", "list"}, `"productDisplayName": "Forerunner 965"`},
		{"calendar json", []string{"--format", "json", "calendar", "--month", "2026-05"}, `"item_type": "workout"`},
	}
	runSuccessCases(t, cfgDir, cases)

	t.Run("workout create and schedule", func(t *testing.T) {
		stdout, stderr, err := runCLI(t, cfgDir, "workouts", "create", "--name", "scheduled", "--step", "warmup 5min", "--schedule", "2026-06-01")
		if err != nil {
			t.Fatalf("runCLI: %v\nstderr:\n%s", err, stderr)
		}
		if !strings.Contains(stdout, "scheduled") {
			t.Fatalf("unexpected output:\n%s", stdout)
		}
	})

	t.Run("workout create from json", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "workout.json")
		if err := os.WriteFile(path, []byte(`{"workoutName":"raw"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		stdout, stderr, err := runCLI(t, cfgDir, "workouts", "create", "--from-json", path)
		if err != nil {
			t.Fatalf("runCLI: %v\nstderr:\n%s", err, stderr)
		}
		if !strings.Contains(stdout, "created") {
			t.Fatalf("unexpected output:\n%s", stdout)
		}
	})

	t.Run("workout export file and force overwrite", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "workout.fit")
		for i := 0; i < 2; i++ {
			args := []string{"workouts", "export", "111", "--out", path}
			if i == 1 {
				args = append(args, "--force")
			}
			_, stderr, err := runCLI(t, cfgDir, args...)
			if err != nil {
				t.Fatalf("runCLI: %v\nstderr:\n%s", err, stderr)
			}
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "FITDATA" {
			t.Fatalf("unexpected FIT: %q", got)
		}
	})

	t.Run("workout export stdout", func(t *testing.T) {
		stdout, stderr, err := runCLI(t, cfgDir, "workouts", "export", "111")
		if err != nil {
			t.Fatalf("runCLI: %v\nstderr:\n%s", err, stderr)
		}
		if stdout != "FITDATA" {
			t.Fatalf("unexpected FIT: %q", stdout)
		}
	})
}

func TestCLI_APIErrorBranches(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rejected", http.StatusBadRequest)
	}))
	defer srv.Close()
	t.Setenv("GARMIN_CONNECTAPI_BASE_URL", srv.URL)

	cfgDir := t.TempDir()
	writeTestSession(t, cfgDir)

	cases := []struct {
		name string
		args []string
	}{
		{"profile", []string{"profile"}},
		{"devices", []string{"devices", "list"}},
		{"records", []string{"records"}},
		{"calendar", []string{"calendar", "--month", "2026-05"}},
		{"courses list", []string{"courses", "list"}},
		{"courses get", []string{"courses", "get", "7"}},
		{"courses import", []string{"courses", "import", "-"}},
		{"courses export", []string{"courses", "export", "7"}},
		{"courses delete", []string{"courses", "delete", "7", "--force"}},
		{"activities weather", []string{"activities", "weather", "123"}},
		{"activities weather json", []string{"--format", "json", "activities", "weather", "123"}},
		{"activities list", []string{"activities", "list"}},
		{"activities get", []string{"activities", "get", "123"}},
		{"activities splits", []string{"activities", "splits", "123"}},
		{"activities export", []string{"activities", "export", "123"}},
		{"activities update", []string{"activities", "update", "123", "--name", "x"}},
		{"activities update type", []string{"activities", "update", "123", "--type", "running"}},
		{"activities delete", []string{"activities", "delete", "123", "--force"}},
		{"gear list", []string{"gear", "list"}},
		{"gear add", []string{"gear", "add", "--name", "x"}},
		{"gear for activity", []string{"gear", "for-activity", "123"}},
		{"workouts list", []string{"workouts", "list"}},
		{"workouts get", []string{"workouts", "get", "123"}},
		{"workouts create", []string{"workouts", "create", "--name", "x", "--step", "warmup 1min"}},
		{"workouts update", []string{"workouts", "update", "123", "--name", "x"}},
		{"workouts schedule", []string{"workouts", "schedule", "123", "--date", "2026-06-01"}},
		{"workouts unschedule", []string{"workouts", "unschedule", "123"}},
		{"workouts export", []string{"workouts", "export", "123"}},
		{"workouts delete", []string{"workouts", "delete", "123", "--force"}},
		{"weight list", []string{"weight", "list", "--days", "2"}},
		{"weight latest", []string{"weight", "latest"}},
		{"weight log", []string{"weight", "log", "75"}},
		{"health summary", []string{"health", "summary", "--date", "2026-05-28"}},
		{"health sleep", []string{"health", "sleep", "--date", "2026-05-28"}},
		{"health heart rate", []string{"health", "heart-rate", "--date", "2026-05-28"}},
		{"health steps", []string{"health", "steps", "--date", "2026-05-28"}},
		{"health stress", []string{"health", "stress", "--date", "2026-05-28"}},
		{"health body battery", []string{"health", "body-battery", "--date", "2026-05-28"}},
		{"health spo2", []string{"health", "spo2", "--date", "2026-05-28"}},
		{"health respiration", []string{"health", "respiration", "--date", "2026-05-28"}},
		{"health intensity", []string{"health", "intensity-minutes", "--date", "2026-05-28"}},
		{"training status", []string{"training", "status", "--date", "2026-05-28"}},
		{"training readiness", []string{"training", "readiness", "--date", "2026-05-28"}},
		{"training vo2max", []string{"training", "vo2max"}},
		{"training hrv", []string{"training", "hrv", "--date", "2026-05-28"}},
		{"training fitness age", []string{"training", "fitness-age", "--date", "2026-05-28"}},
		{"training race predictions", []string{"training", "race-predictions"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := runCLI(t, cfgDir, tc.args...)
			if err == nil {
				t.Fatal("expected API error")
			}
		})
	}
}

func TestCLI_MissingAuthenticationBranches(t *testing.T) {
	t.Setenv("GARMIN_CONNECTAPI_BASE_URL", "http://127.0.0.1:1")
	cfgDir := t.TempDir()
	cases := []struct {
		name string
		args []string
	}{
		{"profile", []string{"profile"}},
		{"devices", []string{"devices", "list"}},
		{"records", []string{"records"}},
		{"calendar", []string{"calendar", "--month", "2026-05"}},
		{"activities weather", []string{"activities", "weather", "1"}},
		{"activities weather json", []string{"--format", "json", "activities", "weather", "1"}},
		{"activities list", []string{"activities", "list"}},
		{"activities get", []string{"activities", "get", "1"}},
		{"activities splits", []string{"activities", "splits", "1"}},
		{"activities export", []string{"activities", "export", "1"}},
		{"activities update", []string{"activities", "update", "1", "--name", "x"}},
		{"activities delete", []string{"activities", "delete", "1", "--force"}},
		{"gear list", []string{"gear", "list"}},
		{"gear get", []string{"gear", "get", "x"}},
		{"gear stats", []string{"gear", "stats", "x"}},
		{"gear add", []string{"gear", "add", "--name", "x"}},
		{"gear retire", []string{"gear", "retire", "x"}},
		{"gear link", []string{"gear", "link", "x", "1"}},
		{"gear for activity", []string{"gear", "for-activity", "1"}},
		{"gear activities", []string{"gear", "activities", "x"}},
		{"gear set default", []string{"gear", "set-default", "x", "--activity-type", "running"}},
		{"workouts list", []string{"workouts", "list"}},
		{"workouts get", []string{"workouts", "get", "1"}},
		{"workouts create", []string{"workouts", "create", "--name", "x", "--step", "warmup 1min"}},
		{"workouts update", []string{"workouts", "update", "1", "--name", "x"}},
		{"workouts schedule", []string{"workouts", "schedule", "1", "--date", "2026-06-01"}},
		{"workouts unschedule", []string{"workouts", "unschedule", "1"}},
		{"workouts export", []string{"workouts", "export", "1"}},
		{"workouts delete", []string{"workouts", "delete", "1", "--force"}},
		{"weight list", []string{"weight", "list", "--days", "2"}},
		{"weight latest", []string{"weight", "latest"}},
		{"weight log", []string{"weight", "log", "75"}},
		{"health summary", []string{"health", "summary", "--date", "2026-05-28"}},
		{"health sleep", []string{"health", "sleep", "--date", "2026-05-28"}},
		{"health stress", []string{"health", "stress", "--date", "2026-05-28"}},
		{"health spo2", []string{"health", "spo2", "--date", "2026-05-28"}},
		{"training status", []string{"training", "status", "--date", "2026-05-28"}},
		{"training vo2max", []string{"training", "vo2max"}},
		{"training fitness age", []string{"training", "fitness-age", "--date", "2026-05-28"}},
		{"training race predictions", []string{"training", "race-predictions"}},
		{"courses list", []string{"courses", "list"}},
		{"courses get", []string{"courses", "get", "1"}},
		{"courses export", []string{"courses", "export", "1"}},
		{"courses delete", []string{"courses", "delete", "1", "--force"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, stderr, err := runCLI(t, cfgDir, tc.args...)
			if err == nil {
				t.Fatal("expected authentication error")
			}
			if !strings.Contains(stderr, "not authenticated") && !strings.Contains(stderr, "not_authenticated") {
				t.Fatalf("unexpected stderr: %s", stderr)
			}
		})
	}
}
