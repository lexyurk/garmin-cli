package courses

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/lexyurk/garmin-cli/internal/auth"
)

type errorWriter struct{}

func (errorWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestSummarizeFallbacksAndActivityNames(t *testing.T) {
	s := summarize(Course{
		CourseID: 3, CourseName: "fallbacks", DistanceInMeters: 12,
		ElevationGainMeters: 4, ElevationLossMeters: 5, ActivityTypePK: 999,
		CourseLines: []CourseLine{{NumberOfPoints: 8}},
	})
	if s.DistanceMeters != 12 || s.ElevationGain != 4 || s.ElevationLoss != 5 || s.Activity != "999" || s.RoutePoints != 8 {
		t.Fatalf("summary=%#v", s)
	}
	if ActivityTypeName(0) != "" || ActivityTypeName(1) != "running" {
		t.Fatal("unexpected activity type names")
	}
	if _, err := ActivityTypeID("swimming"); err == nil {
		t.Fatal("expected unknown activity type error")
	}
	if nilIfEmpty("description") != "description" {
		t.Fatal("non-empty description was lost")
	}
}

func TestCourseReadDeleteAPIs_ValidationAndErrors(t *testing.T) {
	bad, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	defer srv.Close()
	if _, err := List(context.Background(), bad); err == nil {
		t.Fatal("expected list error")
	}
	if _, err := Get(context.Background(), bad, 0); err == nil {
		t.Fatal("expected get validation error")
	}
	if _, err := Get(context.Background(), bad, 1); err == nil {
		t.Fatal("expected get request error")
	}
	if err := Delete(context.Background(), bad, 0); err == nil {
		t.Fatal("expected delete validation error")
	}
	if err := Delete(context.Background(), bad, 1); err == nil {
		t.Fatal("expected delete request error")
	}

	mismatch, mismatchSrv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, Course{CourseID: 2})
	}))
	defer mismatchSrv.Close()
	if _, err := Get(context.Background(), mismatch, 1); err == nil || !strings.Contains(err.Error(), "requested 1, got 2") {
		t.Fatalf("unexpected mismatch error: %v", err)
	}
}

func TestCourseExport_ValidationStatusAndWriterErrors(t *testing.T) {
	ok, okSrv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, "<gpx/>")
	}))
	defer okSrv.Close()
	if err := Export(context.Background(), ok, 0, io.Discard); err == nil {
		t.Fatal("expected id validation error")
	}
	if err := Export(context.Background(), ok, 1, errorWriter{}); err == nil {
		t.Fatal("expected writer error")
	}

	for _, status := range []int{http.StatusUnauthorized, http.StatusBadRequest} {
		c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "rejected", status)
		}))
		err := Export(context.Background(), c, 1, io.Discard)
		srv.Close()
		if err == nil {
			t.Fatalf("expected status %d error", status)
		}
		if status == http.StatusUnauthorized && !errors.Is(err, auth.ErrNotAuthenticated) {
			t.Fatalf("expected auth error, got %v", err)
		}
	}
}

func TestImport_ValidationAndPipelineFailures(t *testing.T) {
	unused, unusedSrv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("unexpected request")
	}))
	defer unusedSrv.Close()
	if _, err := Import(context.Background(), unused, nil, ImportOptions{ActivityType: "running"}); err == nil {
		t.Fatal("expected empty GPX error")
	}
	if _, err := Import(context.Background(), unused, []byte("x"), ImportOptions{ActivityType: "swimming"}); err == nil {
		t.Fatal("expected activity type error")
	}

	t.Run("upload", func(t *testing.T) {
		c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "bad", http.StatusBadRequest)
		}))
		defer srv.Close()
		if _, err := Import(context.Background(), c, []byte("x"), ImportOptions{Filename: "x.gpx", ActivityType: "running"}); err == nil {
			t.Fatal("expected upload error")
		}
	})

	t.Run("empty name", func(t *testing.T) {
		c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, Course{GeoPoints: []GeoPoint{{Latitude: 1}, {Latitude: 2}}})
		}))
		defer srv.Close()
		if _, err := Import(context.Background(), c, []byte("x"), ImportOptions{Filename: ".gpx", ActivityType: "running"}); err == nil {
			t.Fatal("expected empty name error")
		}
	})

	t.Run("create request", func(t *testing.T) {
		c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/course-service/course/import" {
				writeJSON(w, Course{CourseName: "x", GeoPoints: []GeoPoint{{Latitude: 1}, {Latitude: 2}}})
				return
			}
			http.Error(w, "bad", http.StatusBadRequest)
		}))
		defer srv.Close()
		if _, err := Import(context.Background(), c, []byte("x"), ImportOptions{Filename: "x.gpx", ActivityType: "running"}); err == nil {
			t.Fatal("expected create error")
		}
	})

	for _, tc := range []struct {
		name    string
		created Course
		verify  Course
	}{
		{"missing id", Course{}, Course{}},
		{"name mismatch", Course{CourseID: 5}, Course{CourseID: 5, CourseName: "other"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, srv := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/course-service/course/import":
					writeJSON(w, Course{CourseName: "expected", GeoPoints: []GeoPoint{{Latitude: 1}, {Latitude: 2}}})
				case "/course-service/course":
					writeJSON(w, tc.created)
				default:
					writeJSON(w, tc.verify)
				}
			}))
			defer srv.Close()
			if _, err := Import(context.Background(), c, []byte("x"), ImportOptions{Filename: "x.gpx", ActivityType: "running"}); err == nil {
				t.Fatal("expected import error")
			}
		})
	}
}

func TestBuildPayloadRejectsTooFewPoints(t *testing.T) {
	if _, err := BuildPayload(Course{}, "empty", 1, "", nil); err == nil {
		t.Fatal("expected too-few-points error")
	}
}
