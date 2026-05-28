package main

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestCLI_ValidationErrors covers command argument/flag validation that fails
// before any network call, exercising the early-return branches in the new
// command bodies. A guard base URL ensures no case can reach the live API.
func TestCLI_ValidationErrors(t *testing.T) {
	t.Setenv("GARMIN_CONNECTAPI_BASE_URL", "http://127.0.0.1:1")
	cfgDir := t.TempDir()
	writeTestSession(t, cfgDir)

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"gear list all+retired", []string{"gear", "list", "--all", "--retired"}, "either --all or --retired"},
		{"gear link neither id nor last", []string{"gear", "link", "Pegasus"}, "activity id or --last"},
		{"gear link both id and last", []string{"gear", "link", "Pegasus", "123", "--last"}, "activity id or --last"},
		{"gear add missing name", []string{"gear", "add"}, "name"},
		{"gear set-default missing type", []string{"gear", "set-default", "Pegasus"}, "activity-type"},
		{"workouts list bad limit", []string{"workouts", "list", "--limit", "0"}, "--limit must be > 0"},
		{"workouts get bad id", []string{"workouts", "get", "abc"}, "invalid workout id"},
		{"workouts delete bad id", []string{"workouts", "delete", "abc"}, "invalid workout id"},
		{"workouts create no steps", []string{"workouts", "create", "--name", "x"}, "at least one --step"},
		{"workouts create bad json", []string{"workouts", "create", "--from-json", filepath.Join(t.TempDir(), "nope.json")}, "no such file"},
		{"workouts update no fields", []string{"workouts", "update", "111"}, "at least one of --name"},
		{"workouts schedule no date", []string{"workouts", "schedule", "111"}, "--date is required"},
		{"activities update no fields", []string{"activities", "update", "123"}, "at least one of --name"},
		{"activities delete no force", []string{"activities", "delete", "123"}, "aborted"},
		{"weight log bad number", []string{"weight", "log", "abc"}, "invalid weight"},
		{"calendar bad month", []string{"calendar", "--month", "nope"}, "invalid --month"},
		{"training fitness-age bad date", []string{"training", "fitness-age", "--date", "nope"}, "invalid --date"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := runCLI(t, cfgDir, tc.args...)
			if err == nil {
				t.Fatalf("expected an error")
			}
			if tc.want != "" && !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected error to contain %q, got: %v", tc.want, err)
			}
		})
	}
}
