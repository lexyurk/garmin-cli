package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCoursesRegisteredAndValidation(t *testing.T) {
	root := NewRootCmd("dev")
	found := false
	for _, cmd := range root.Commands() {
		if cmd.Name() == "courses" {
			found = true
		}
	}
	if !found {
		t.Fatal("courses command not registered")
	}

	for _, tc := range []struct {
		args     []string
		contains string
	}{
		{[]string{"get", "nope"}, "invalid course id"},
		{[]string{"delete", "7"}, "pass --force"},
		{[]string{"import", "-", "--point", "bad"}, "invalid course point"},
		{[]string{"import", "-", "--replace", "-1"}, "--replace must be a positive"},
	} {
		cmd := NewCoursesCmd(&globalOptions{})
		cmd.SetIn(strings.NewReader("<gpx/>"))
		cmd.SetArgs(tc.args)
		err := cmd.Execute()
		if err == nil || !strings.Contains(err.Error(), tc.contains) {
			t.Fatalf("args=%v err=%v, want %q", tc.args, err, tc.contains)
		}
	}
}

func TestCoursesExportRefusesOverwriteBeforeNetwork(t *testing.T) {
	path := filepath.Join(t.TempDir(), "course.gpx")
	if err := os.WriteFile(path, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := newCoursesExportCmd(&globalOptions{})
	cmd.SetArgs([]string{"7", "--out", path})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("unexpected err: %v", err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "keep" {
		t.Fatalf("existing output changed: %q", data)
	}
}

func TestCoursesExportRefusesTerminal(t *testing.T) {
	original := courseIsTerminal
	courseIsTerminal = func(int) bool { return true }
	t.Cleanup(func() { courseIsTerminal = original })
	f, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cmd := newCoursesExportCmd(&globalOptions{})
	cmd.SetOut(f)
	cmd.SetArgs([]string{"7"})
	err = cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "refusing to write GPX to terminal") {
		t.Fatalf("unexpected err: %v", err)
	}
}

func TestCoursesImportSafeReplaceOrdering(t *testing.T) {
	cfg := t.TempDir()
	writeTestSession(t, cfg)
	var calls []string
	mux := http.NewServeMux()
	mux.HandleFunc("/course-service/course/import", func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "import")
		_ = r.ParseMultipartForm(1 << 20)
		writeCourseJSON(w, map[string]any{"courseName": "Imported", "geoPoints": []map[string]any{{"latitude": 1, "longitude": 1}, {"latitude": 1.01, "longitude": 1.01}}})
	})
	mux.HandleFunc("/course-service/course", func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "create")
		writeCourseJSON(w, map[string]any{"courseId": 8, "courseName": "Imported"})
	})
	mux.HandleFunc("/course-service/course/8", func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "verify")
		writeCourseJSON(w, map[string]any{"courseId": 8, "courseName": "Imported", "distanceMeter": 1000})
	})
	mux.HandleFunc("/course-service/course/7", func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, "delete-old")
		w.WriteHeader(http.StatusNoContent)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	t.Setenv("GARMIN_CONNECTAPI_BASE_URL", srv.URL)

	cmd := NewRootCmd("dev")
	cmd.SetIn(strings.NewReader("<gpx/>"))
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"--config-dir", cfg, "--format", "json", "courses", "import", "-", "--replace", "7", "--force"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v stderr=%s", err, errOut.String())
	}
	if !reflect.DeepEqual(calls, []string{"import", "create", "verify", "delete-old"}) {
		t.Fatalf("unsafe ordering: %v", calls)
	}
	if !strings.Contains(out.String(), `"id": 8`) {
		t.Fatalf("unexpected output: %s", out.String())
	}
}

func TestCoursesImportFailedVerificationKeepsOld(t *testing.T) {
	cfg := t.TempDir()
	writeTestSession(t, cfg)
	deleted := false
	mux := http.NewServeMux()
	mux.HandleFunc("/course-service/course/import", func(w http.ResponseWriter, r *http.Request) {
		writeCourseJSON(w, map[string]any{"courseName": "Imported", "geoPoints": []map[string]any{{"latitude": 1, "longitude": 1}, {"latitude": 1.01, "longitude": 1.01}}})
	})
	mux.HandleFunc("/course-service/course", func(w http.ResponseWriter, r *http.Request) {
		writeCourseJSON(w, map[string]any{"courseId": 8, "courseName": "Imported"})
	})
	mux.HandleFunc("/course-service/course/8", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "no readback", http.StatusInternalServerError)
	})
	mux.HandleFunc("/course-service/course/7", func(w http.ResponseWriter, r *http.Request) { deleted = true; w.WriteHeader(http.StatusNoContent) })
	srv := httptest.NewServer(mux)
	defer srv.Close()
	t.Setenv("GARMIN_CONNECTAPI_BASE_URL", srv.URL)

	cmd := NewRootCmd("dev")
	cmd.SetIn(strings.NewReader("<gpx/>"))
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"--config-dir", cfg, "courses", "import", "-", "--replace", "7", "--force"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "verification failed") {
		t.Fatalf("unexpected err: %v", err)
	}
	if deleted {
		t.Fatal("old course was deleted after verification failure")
	}
}

func TestCoursesCLIReadExportDelete(t *testing.T) {
	cfg := t.TempDir()
	writeTestSession(t, cfg)
	deleted := false
	mux := http.NewServeMux()
	mux.HandleFunc("/course-service/course", func(w http.ResponseWriter, r *http.Request) {
		writeCourseJSON(w, []map[string]any{{"courseId": 7, "courseName": "Seven", "distanceInMeters": 5000, "activityType": map[string]any{"typeKey": "running"}}})
	})
	mux.HandleFunc("/course-service/course/7", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleted = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeCourseJSON(w, map[string]any{"courseId": 7, "courseName": "Seven", "distanceMeter": 5000, "elevationGainMeter": 12, "activityTypePk": 1})
	})
	mux.HandleFunc("/course-service/course/gpx/7", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte("<gpx/>")) })
	srv := httptest.NewServer(mux)
	defer srv.Close()
	t.Setenv("GARMIN_CONNECTAPI_BASE_URL", srv.URL)

	out, _, err := runCLI(t, cfg, "courses", "list")
	if err != nil || !strings.Contains(out, "Seven") {
		t.Fatalf("list out=%q err=%v", out, err)
	}
	if strings.Contains(out, "points") {
		t.Fatalf("list must not claim point counts unavailable from the list endpoint: %q", out)
	}
	jsonOut, _, err := runCLI(t, cfg, "--format", "json", "courses", "list")
	if err != nil {
		t.Fatalf("json list err=%v", err)
	}
	var listed []map[string]any
	if err := json.Unmarshal([]byte(jsonOut), &listed); err != nil {
		t.Fatalf("decode json list: %v (%q)", err, jsonOut)
	}
	if len(listed) != 1 {
		t.Fatalf("json list=%#v", listed)
	}
	if _, ok := listed[0]["route_points"]; ok {
		t.Fatalf("json list must omit unknown route point count: %#v", listed[0])
	}
	if _, ok := listed[0]["course_points"]; ok {
		t.Fatalf("json list must omit unknown course point count: %#v", listed[0])
	}
	out, _, err = runCLI(t, cfg, "courses", "get", "7")
	if err != nil || !strings.Contains(out, "https://connect.garmin.com/modern/course/7") {
		t.Fatalf("get out=%q err=%v", out, err)
	}
	outPath := filepath.Join(t.TempDir(), "nested", "seven.gpx")
	_, stderr, err := runCLI(t, cfg, "courses", "export", "7", "--out", outPath)
	if err != nil || !strings.Contains(stderr, "downloaded") {
		t.Fatalf("export stderr=%q err=%v", stderr, err)
	}
	data, _ := os.ReadFile(outPath)
	if string(data) != "<gpx/>" {
		t.Fatalf("export data=%q", data)
	}
	deleteOut, _, err := runCLI(t, cfg, "--format", "json", "courses", "delete", "7", "--force")
	if err != nil || !deleted {
		t.Fatalf("delete err=%v deleted=%v", err, deleted)
	}
	var deletedJSON struct {
		ID     int64  `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(deleteOut), &deletedJSON); err != nil {
		t.Fatalf("delete output is not JSON: %v (%q)", err, deleteOut)
	}
	if deletedJSON.ID != 7 || deletedJSON.Status != "deleted" {
		t.Fatalf("delete JSON=%#v", deletedJSON)
	}
}

func writeCourseJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
