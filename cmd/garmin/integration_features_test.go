package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// newFeatureAPIServer mocks the Connect API endpoints used by the gear,
// workouts, weight, devices, records, calendar, today, and extended health/
// training/activities commands.
func newFeatureAPIServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	gearObj := map[string]any{
		"uuid": "u1", "gearPk": 1, "displayName": "Pegasus 40",
		"gearMakeName": "Nike", "gearModelName": "Pegasus 40",
		"gearTypeName": "Shoes", "gearStatusName": "active",
		"maximumMeters": 800000, "dateBegin": "2026-01-01T00:00:00.0",
	}

	mux.HandleFunc("/userprofile-service/socialProfile", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"profileId": 987, "displayName": "runner", "fullName": "Sam Run", "userName": "sam", "location": "Berlin"})
	})

	// --- gear ---
	mux.HandleFunc("/gear-service/gear/filterGear", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []any{gearObj})
	})
	mux.HandleFunc("/gear-service/gear", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"uuid": "u2", "displayName": "New Shoe", "gearTypeName": "Shoes", "gearStatusName": "active"})
	})
	mux.HandleFunc("/gear-service/gear/", func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		switch {
		case strings.Contains(p, "/stats/"):
			writeJSON(w, map[string]any{"totalDistance": 123456.0, "totalActivities": 42})
		case strings.Contains(p, "/activityType/"), strings.Contains(p, "/link/"), strings.Contains(p, "/unlink/"):
			w.WriteHeader(http.StatusOK)
		default: // PUT /gear-service/gear/{uuid} (retire/restore)
			writeJSON(w, map[string]any{"uuid": "u1", "displayName": "Pegasus 40", "gearStatusName": "retired"})
		}
	})

	// --- activities ---
	mux.HandleFunc("/activitylist-service/activities/search/activities", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("start") != "0" {
			writeJSON(w, []any{})
			return
		}
		writeJSON(w, []any{map[string]any{
			"activityId": 123, "activityName": "Morning Run", "startTimeLocal": "2026-05-28 07:00:00",
			"activityType": map[string]any{"typeKey": "running"}, "distance": 5000, "duration": 1500, "calories": 321, "averageHR": 140,
		}})
	})
	mux.HandleFunc("/activitylist-service/activities/", func(w http.ResponseWriter, r *http.Request) {
		// {uuid}/gear
		writeJSON(w, []any{map[string]any{
			"activityId": 123, "activityName": "Morning Run", "startTimeLocal": "2026-05-28 07:00:00",
			"activityType": map[string]any{"typeKey": "running"}, "distance": 5000, "duration": 1500,
		}})
	})
	mux.HandleFunc("/activity-service/activity/activityTypes", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []any{
			map[string]any{"typeId": 1, "typeKey": "running", "parentTypeId": 17},
			map[string]any{"typeId": 2, "typeKey": "cycling", "parentTypeId": 17},
		})
	})
	mux.HandleFunc("/activity-service/activity/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/weather"):
			writeJSON(w, map[string]any{"temp": 12.0, "apparentTemp": 10.0, "relativeHumidity": 80, "windSpeed": 5.5, "windDirectionCompassPoint": "NW", "weatherTypeDTO": map[string]any{"desc": "Cloudy"}})
		case r.Method == http.MethodPut, r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			writeJSON(w, map[string]any{
				"activityName": "Morning Run", "startTimeLocal": "2026-05-28 07:00:00",
				"activityType": map[string]any{"typeKey": "running"}, "distance": 5000, "duration": 1500, "calories": 321, "averageHR": 140, "maxHR": 180,
			})
		}
	})

	// --- workouts ---
	mux.HandleFunc("/workout-service/workouts", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []any{map[string]any{
			"workoutId": 111, "workoutName": "Intervals", "sportType": map[string]any{"sportTypeId": 1, "sportTypeKey": "running"},
			"estimatedDurationInSecs": 3600, "estimatedDistanceInMeters": 10000,
		}})
	})
	mux.HandleFunc("/workout-service/workout", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"workoutId": 999, "workoutName": "4x800m", "sportType": map[string]any{"sportTypeKey": "running"}})
	})
	mux.HandleFunc("/workout-service/workout/FIT/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("FITDATA"))
	})
	mux.HandleFunc("/workout-service/workout/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSON(w, map[string]any{
			"workoutId": 111, "workoutName": "Intervals", "description": "speed",
			"sportType": map[string]any{"sportTypeKey": "running"}, "estimatedDurationInSecs": 3600,
			"workoutSegments": []any{map[string]any{"workoutSteps": []any{
				map[string]any{"stepType": map[string]any{"stepTypeKey": "warmup"}},
				map[string]any{"stepType": map[string]any{"stepTypeKey": "cooldown"}},
			}}},
		})
	})
	mux.HandleFunc("/workout-service/schedule/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSON(w, map[string]any{"workoutScheduleId": 9001})
	})

	// --- weight ---
	mux.HandleFunc("/weight-service/weight/dateRange", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"dateWeightList": []any{
			map[string]any{"samplePk": 9, "calendarDate": "2026-05-28", "weight": 75000, "bmi": 22.5, "bodyFat": 15.2},
		}})
	})
	mux.HandleFunc("/weight-service/user-weight", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// --- devices / records / predictions / fitness age ---
	mux.HandleFunc("/device-service/deviceregistration/devices", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []any{map[string]any{"deviceId": 111, "productDisplayName": "Forerunner 965", "serialNumber": "abc", "partNumber": "006-X"}})
	})
	mux.HandleFunc("/personalrecord-service/personalrecord/prs/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []any{map[string]any{"typeId": 3, "value": 1350, "activityId": 7, "activityName": "5k TT"}})
	})
	mux.HandleFunc("/metrics-service/metrics/racepredictions/latest/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"calendarDate": "2026-05-28", "time5K": 1350, "time10K": 2820, "timeHalfMarathon": 6300, "timeMarathon": 13500})
	})
	mux.HandleFunc("/fitnessage-service/fitnessage/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"chronologicalAge": 38, "fitnessAge": 32.5, "achievableFitnessAge": 30, "previousFitnessAge": 33, "components": map[string]any{"rhr": map[string]any{"value": 52}}})
	})

	// --- today (daily summary + training) ---
	mux.HandleFunc("/usersummary-service/usersummary/daily/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"calendarDate": r.URL.Query().Get("calendarDate"), "totalSteps": 1234, "restingHeartRate": 52,
			"averageStressLevel": 23, "bodyBatteryMostRecentValue": 38,
		})
	})
	mux.HandleFunc("/mobile-gateway/usersummary/trainingstatus/latest/", func(w http.ResponseWriter, r *http.Request) {
		date := strings.TrimPrefix(r.URL.Path, "/mobile-gateway/usersummary/trainingstatus/latest/")
		writeJSON(w, map[string]any{"mostRecentTrainingStatus": map[string]any{"payload": map[string]any{"latestTrainingStatusData": map[string]any{
			date: map[string]any{"trainingStatusFeedbackPhrase": "Productive", "trainingStatus": 1, "weeklyTrainingLoad": 123, "loadLevelTrend": "STABLE"},
		}}}})
	})
	mux.HandleFunc("/metrics-service/metrics/trainingreadiness/", func(w http.ResponseWriter, r *http.Request) {
		date := strings.TrimPrefix(r.URL.Path, "/metrics-service/metrics/trainingreadiness/")
		writeJSON(w, []any{map[string]any{"calendarDate": date, "timestamp": date + "T09:00:00.0", "level": "HIGH", "score": 80}})
	})

	// --- health depth ---
	mux.HandleFunc("/wellness-service/wellness/daily/spo2/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"calendarDate": "2026-05-28", "averageSpO2": 96, "lowestSpO2": 92, "latestSpO2": 95})
	})
	mux.HandleFunc("/wellness-service/wellness/daily/respiration/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"calendarDate": "2026-05-28", "avgWakingRespirationValue": 14.5, "highestRespirationValue": 22, "lowestRespirationValue": 10})
	})
	mux.HandleFunc("/wellness-service/wellness/daily/im/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"calendarDate": "2026-05-28", "moderateValue": 20, "vigorousValue": 35, "weeklyGoal": 150})
	})

	// --- calendar ---
	mux.HandleFunc("/calendar-service/year/", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"calendarItems": []any{
			map[string]any{"date": "2026-05-03", "itemType": "workout", "title": "Intervals", "workoutId": 111},
			map[string]any{"date": "2026-05-05", "itemType": "activity", "title": "Run", "activityId": 123},
		}})
	})

	return httptest.NewServer(mux)
}

func TestCLI_FeatureFlows(t *testing.T) {
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
		{"profile", []string{"profile"}, "runner"},
		{"profile json", []string{"--format", "json", "profile"}, `"profile_id": 987`},
		{"gear list", []string{"gear", "list"}, "Pegasus 40"},
		{"gear list stats", []string{"gear", "list", "--all", "--stats"}, "total_km"},
		{"gear get by name", []string{"gear", "get", "Pegasus 40"}, "Nike"},
		{"gear stats", []string{"gear", "stats", "Pegasus 40"}, "total_km"},
		{"gear add", []string{"gear", "add", "--name", "New Shoe", "--make", "Nike"}, "Gear added"},
		{"gear retire", []string{"gear", "retire", "Pegasus 40"}, "retired"},
		{"gear restore", []string{"gear", "restore", "u1"}, "Gear restored"},
		{"gear link last", []string{"gear", "link", "Pegasus 40", "--last"}, "linked"},
		{"gear unlink id", []string{"gear", "unlink", "Pegasus 40", "123"}, "unlinked"},
		{"gear for-activity", []string{"gear", "for-activity", "123"}, "Pegasus 40"},
		{"gear activities", []string{"gear", "activities", "Pegasus 40"}, "Morning Run"},
		{"gear set-default", []string{"gear", "set-default", "Pegasus 40", "--activity-type", "running"}, "default-set"},
		{"gear clear-default", []string{"gear", "clear-default", "Pegasus 40", "--activity-type", "running"}, "default-cleared"},
		{"workouts list", []string{"workouts", "list"}, "Intervals"},
		{"workouts get", []string{"workouts", "get", "111"}, "steps"},
		{"workouts create", []string{"workouts", "create", "--name", "4x800m", "--step", "warmup 10min", "--step", "4x(interval 800m; recovery 2min)", "--step", "cooldown 5min"}, "created"},
		{"workouts create json", []string{"--format", "json", "workouts", "create", "--name", "x", "--step", "warmup 10min"}, `"workout_id": 999`},
		{"workouts update", []string{"workouts", "update", "111", "--name", "renamed"}, "updated"},
		{"workouts delete", []string{"workouts", "delete", "111", "--force"}, "deleted"},
		{"workouts schedule", []string{"workouts", "schedule", "111", "--date", "2026-06-01"}, "scheduled"},
		{"workouts unschedule", []string{"workouts", "unschedule", "9001"}, "unscheduled"},
		{"weight log", []string{"weight", "log", "74.5"}, "logged"},
		{"weight list", []string{"weight", "list", "--days", "30"}, "weight_kg"},
		{"weight latest", []string{"weight", "latest"}, "weight_kg"},
		{"devices list", []string{"devices", "list"}, "Forerunner 965"},
		{"records", []string{"records"}, "5 km"},
		{"records json", []string{"--format", "json", "records"}, "typeId"},
		{"training fitness-age", []string{"training", "fitness-age"}, "fitness_age"},
		{"training race-predictions", []string{"training", "race-predictions"}, "marathon"},
		{"activities update", []string{"activities", "update", "123", "--name", "Tempo", "--type", "running"}, "updated"},
		{"activities delete force", []string{"activities", "delete", "123", "--force"}, "deleted"},
		{"activities weather", []string{"activities", "weather", "123"}, "Cloudy"},
		{"activities weather json", []string{"--format", "json", "activities", "weather", "123"}, "weatherTypeDTO"},
		{"calendar", []string{"calendar", "--month", "2026-05"}, "Intervals"},
		{"calendar type filter", []string{"calendar", "--month", "2026-05", "--type", "workout"}, "workout"},
		{"health spo2", []string{"health", "spo2", "--date", "2026-05-28"}, "average"},
		{"health respiration", []string{"health", "respiration", "--date", "2026-05-28"}, "avg_waking"},
		{"health intensity-minutes", []string{"health", "intensity-minutes", "--date", "2026-05-28"}, "moderate"},
		{"today", []string{"today"}, "training_status"},
		{"today json", []string{"--format", "json", "today"}, "last_activity"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := runCLI(t, cfgDir, tc.args...)
			if err != nil {
				t.Fatalf("err: %v\nstderr:\n%s", err, stderr)
			}
			if !strings.Contains(stdout, tc.want) {
				t.Fatalf("expected %q in output, got:\n%s", tc.want, stdout)
			}
		})
	}
}
