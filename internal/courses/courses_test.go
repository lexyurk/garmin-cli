package courses

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/lexyurk/garmin-cli/internal/auth"
	"github.com/lexyurk/garmin-cli/internal/client"
)

func testClient(t *testing.T, h http.Handler) (*client.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(h)
	sess := &auth.Session{OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "test", ExpiresAt: time.Now().Add(time.Hour).Unix()}}
	c := client.NewWithSession("ignored", "", sess, client.Options{HTTPClient: srv.Client(), BaseURL: srv.URL})
	return c, srv
}

func float(v float64) *float64 { return &v }

func TestBuildPayloadMathAndNearestCoursePoint(t *testing.T) {
	parsed := Course{GeoPoints: []GeoPoint{
		{Latitude: 52, Longitude: 4, Elevation: float(7)},
		{Latitude: 52.001, Longitude: 4, Elevation: float(9)},
		{Latitude: 52.002, Longitude: 4, Elevation: nil},
	}}
	payload, err := BuildPayload(parsed, "Route", 1, "desc", []PointSpec{{Type: "WATER", Distance: 115, Name: "Tap"}})
	if err != nil {
		t.Fatal(err)
	}
	distance := payload["distanceMeter"].(float64)
	if math.Abs(distance-222.39) > .5 {
		t.Fatalf("unexpected haversine distance: %.3f", distance)
	}
	if payload["rulePK"] != 2 || payload["sourceTypeId"] != 3 || payload["coordinateSystem"] != "WGS84" {
		t.Fatalf("missing Garmin payload constants: %#v", payload)
	}
	line := payload["courseLines"].([]CourseLine)[0]
	if line.NumberOfPoints != 3 || line.DistanceInMeters != distance || line.Points[2].Elevation == nil || *line.Points[2].Elevation != 0 {
		t.Fatalf("bad course line: %#v", line)
	}
	cp := payload["coursePoints"].([]CoursePoint)[0]
	if cp.Name != "Tap" || cp.CoursePointType != "WATER" || cp.Lat != 52.001 || cp.Elevation != 9 || math.Abs(cp.Distance-111.19) > .5 {
		t.Fatalf("course point not snapped to nearest geo point: %#v", cp)
	}
	bbox := payload["boundingBox"].(map[string]any)
	if bbox["lowerLeftLatIsSet"] != true || bbox["upperRightLongIsSet"] != true {
		t.Fatalf("bad bbox flags: %#v", bbox)
	}
}

func TestParsePointSpecAndDistance(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want float64
	}{{"12", 12}, {"12m", 12}, {"1.25km", 1250}} {
		got, err := ParseDistance(tc.raw)
		if err != nil || got != tc.want {
			t.Fatalf("ParseDistance(%q)=(%v,%v), want %v", tc.raw, got, err, tc.want)
		}
	}
	p, err := ParsePointSpec(" water | 1.25km | North tap ")
	if err != nil || p != (PointSpec{Type: "WATER", Distance: 1250, Name: "North tap"}) {
		t.Fatalf("unexpected point: %#v %v", p, err)
	}
	for _, raw := range []string{"WATER|12", "|12|tap", "WATER|-1m|tap", "WATER|nope|tap", "WATER|NaN|tap", "WATER|Inf|tap", "WATER|-Inf|tap"} {
		if _, err := ParsePointSpec(raw); err == nil {
			t.Fatalf("expected %q to fail", raw)
		}
	}
}

func TestBuildPayloadRejectsCoursePointOutsideRoute(t *testing.T) {
	parsed := Course{GeoPoints: []GeoPoint{{Latitude: 52, Longitude: 4}, {Latitude: 52.001, Longitude: 4}}}
	for _, spec := range []PointSpec{
		{Type: "WATER", Distance: -1, Name: "Before start"},
		{Type: "WATER", Distance: 200, Name: "Past finish"},
		{Type: "WATER", Distance: math.NaN(), Name: "Broken"},
		{Type: "WATER", Distance: math.Inf(1), Name: "Infinite"},
	} {
		_, err := BuildPayload(parsed, "Route", 1, "", []PointSpec{spec})
		if err == nil {
			t.Fatalf("expected point %#v to fail", spec)
		}
		if !strings.Contains(err.Error(), spec.Name) || !strings.Contains(err.Error(), "distance") || !strings.Contains(err.Error(), "route") {
			t.Fatalf("error should name point, distance, and route length: %v", err)
		}
	}
}

func TestImportMultipartCreateVerifyOrdering(t *testing.T) {
	var order []string
	mux := http.NewServeMux()
	mux.HandleFunc("/course-service/course/import", func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "import")
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		mr, err := r.MultipartReader()
		if err != nil {
			t.Fatal(err)
		}
		part, err := mr.NextPart()
		if err != nil {
			t.Fatal(err)
		}
		if part.FormName() != "file" || part.FileName() != "route.gpx" || part.Header.Get("Content-Type") != "application/gpx+xml" {
			t.Fatalf("bad multipart part: %#v", part)
		}
		data, _ := io.ReadAll(part)
		if string(data) != "<gpx/>" {
			t.Fatalf("bad GPX: %q", data)
		}
		writeJSON(w, Course{CourseName: "Imported", GeoPoints: []GeoPoint{{Latitude: 52, Longitude: 4}, {Latitude: 52.001, Longitude: 4}}})
	})
	mux.HandleFunc("/course-service/course", func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "create")
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			t.Fatalf("content type=%q", r.Header.Get("Content-Type"))
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["courseName"] != "Named" || body["activityTypePk"] != float64(6) {
			t.Fatalf("bad create payload: %#v", body)
		}
		writeJSON(w, Course{CourseID: 42, CourseName: "Named"})
	})
	mux.HandleFunc("/course-service/course/42", func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "verify")
		writeJSON(w, Course{CourseID: 42, CourseName: "Named", DistanceMeter: 111, ElevationGainMeter: 5, GeoPoints: make([]GeoPoint, 2)})
	})
	c, srv := testClient(t, mux)
	defer srv.Close()
	got, err := Import(context.Background(), c, []byte("<gpx/>"), ImportOptions{Filename: "route.gpx", Name: "Named", ActivityType: "trail_running"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 42 || got.URL != "https://connect.garmin.com/modern/course/42" || got.RoutePoints != 2 {
		t.Fatalf("unexpected summary: %#v", got)
	}
	if !reflect.DeepEqual(order, []string{"import", "create", "verify"}) {
		t.Fatalf("order=%v", order)
	}
}

func TestImportVerificationFailureDoesNotDeleteAnything(t *testing.T) {
	var methods []string
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.Path)
		switch r.URL.Path {
		case "/course-service/course/import":
			writeJSON(w, Course{CourseName: "x", GeoPoints: []GeoPoint{{Latitude: 1, Longitude: 1}, {Latitude: 1.1, Longitude: 1.1}}})
		case "/course-service/course":
			writeJSON(w, Course{CourseID: 99, CourseName: "x"})
		case "/course-service/course/99":
			http.Error(w, "broken", http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected endpoint: %s %s", r.Method, r.URL.Path)
		}
	})
	c, srv := testClient(t, h)
	defer srv.Close()
	_, err := Import(context.Background(), c, []byte("x"), ImportOptions{Filename: "x.gpx", ActivityType: "running"})
	if err == nil || !strings.Contains(err.Error(), "verification failed") {
		t.Fatalf("unexpected err: %v", err)
	}
	for _, call := range methods {
		if strings.HasPrefix(call, "DELETE ") {
			t.Fatalf("unsafe delete after failed verification: %v", methods)
		}
	}
}

func TestListGetExportDeleteEndpoints(t *testing.T) {
	var deleted bool
	mux := http.NewServeMux()
	mux.HandleFunc("/course-service/course", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, []Course{{CourseID: 7, CourseName: "Seven", DistanceInMeters: 5000, ActivityType: ActivityType{TypeKey: "running"}}})
	})
	mux.HandleFunc("/course-service/course/7", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleted = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		writeJSON(w, Course{CourseID: 7, CourseName: "Seven", DistanceMeter: 5000, ActivityTypePK: 1})
	})
	mux.HandleFunc("/course-service/course/gpx/7", func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, "<gpx/>") })
	c, srv := testClient(t, mux)
	defer srv.Close()
	list, err := List(context.Background(), c)
	if err != nil || len(list) != 1 || list[0].DistanceMeters != 5000 {
		t.Fatalf("list=%#v err=%v", list, err)
	}
	got, err := Get(context.Background(), c, 7)
	if err != nil || got.Activity != "running" {
		t.Fatalf("get=%#v err=%v", got, err)
	}
	var out bytes.Buffer
	if err := Export(context.Background(), c, 7, &out); err != nil || out.String() != "<gpx/>" {
		t.Fatalf("export=%q err=%v", out.String(), err)
	}
	if err := Delete(context.Background(), c, 7); err != nil || !deleted {
		t.Fatalf("delete err=%v deleted=%v", err, deleted)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
