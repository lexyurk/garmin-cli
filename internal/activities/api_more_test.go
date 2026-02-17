package activities

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lexyurk/garmin-cli/internal/auth"
	"github.com/lexyurk/garmin-cli/internal/client"
)

func newAuthedTestClient(srv *httptest.Server) *client.Client {
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}
	return client.NewWithSession("ignored", "default", sess, client.Options{HTTPClient: srv.Client(), BaseURL: srv.URL})
}

func TestList_PaginatesAndRespectsLimit(t *testing.T) {
	var gotQueries []url.Values

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activitylist-service/activities/search/activities" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		q := r.URL.Query()
		gotQueries = append(gotQueries, q)

		start, _ := strconv.Atoi(q.Get("start"))
		limit, _ := strconv.Atoi(q.Get("limit"))
		if limit != 50 {
			t.Fatalf("expected page limit=50, got %d", limit)
		}

		var items []ListItem
		switch start {
		case 0:
			items = make([]ListItem, 50)
			for i := range items {
				items[i] = ListItem{
					ActivityID:     int64(1000 + i),
					ActivityName:   "Run",
					StartTimeLocal: "2026-02-16 07:00:00",
					ActivityType:   TypeInfo{TypeKey: "running"},
					Distance:       5000,
					Duration:       1500,
					Calories:       300,
					AverageHR:      140,
				}
			}
		case 50:
			items = make([]ListItem, 15)
			for i := range items {
				items[i] = ListItem{
					ActivityID:     int64(2000 + i),
					ActivityName:   "Ride",
					StartTimeLocal: "2026-02-15 07:00:00",
					ActivityType:   TypeInfo{TypeKey: "cycling"},
					Distance:       20000,
					Duration:       3600,
				}
			}
		default:
			items = []ListItem{}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(items)
	}))
	defer srv.Close()

	c := newAuthedTestClient(srv)
	out, err := List(context.Background(), c, 60, "", "", "")
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if len(out) != 60 {
		t.Fatalf("expected 60 items, got %d", len(out))
	}
	if out[0].Date != "2026-02-16" {
		t.Fatalf("expected date derived from StartTimeLocal, got %q", out[0].Date)
	}
	if len(gotQueries) != 2 {
		t.Fatalf("expected 2 page requests, got %d", len(gotQueries))
	}
	if gotQueries[0].Get("start") != "0" || gotQueries[1].Get("start") != "50" {
		t.Fatalf("unexpected paging queries: %#v", gotQueries)
	}
}

func TestList_ValidatesLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request")
	}))
	defer srv.Close()

	c := newAuthedTestClient(srv)
	if _, err := List(context.Background(), c, 0, "", "", ""); err == nil {
		t.Fatalf("expected error for limit=0")
	}
}

func TestParseDate_AndPassesFilters(t *testing.T) {
	if _, err := parseDate("2026-99-99"); err == nil {
		t.Fatalf("expected error")
	}
	got, err := parseDate("2026-02-16")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "2026-02-16" {
		t.Fatalf("unexpected parsed date: %q", got)
	}

	a := Summary{Date: "2026-02-16", Type: "Running"}
	if passesFilters(a, "2026-02-17", "", "") {
		t.Fatalf("expected after filter to reject")
	}
	if passesFilters(a, "", "2026-02-15", "") {
		t.Fatalf("expected before filter to reject")
	}
	if !passesFilters(a, "", "", "run") {
		t.Fatalf("expected type substring filter to pass")
	}
	if passesFilters(a, "", "", "cycling") {
		t.Fatalf("expected type filter to reject")
	}
}

func TestGetRaw(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activity-service/activity/123" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"activityName":"Test"}`))
	}))
	defer srv.Close()

	c := newAuthedTestClient(srv)
	raw, err := GetRaw(context.Background(), c, 123)
	if err != nil {
		t.Fatalf("GetRaw error: %v", err)
	}
	if raw["activityName"] != "Test" {
		t.Fatalf("unexpected raw: %#v", raw)
	}
}

func TestExport_UnsupportedType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request")
	}))
	defer srv.Close()

	c := newAuthedTestClient(srv)
	var b bytes.Buffer
	if err := Export(context.Background(), c, 1, ExportType("zip"), &b); err == nil {
		t.Fatalf("expected error")
	}
}

func TestExport_Non2xxError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("bad"))
	}))
	defer srv.Close()

	c := newAuthedTestClient(srv)
	var b bytes.Buffer
	err := Export(context.Background(), c, 1, ExportGPX, &b)
	if err == nil {
		t.Fatalf("expected error")
	}
	if errors.Is(err, auth.ErrNotAuthenticated) {
		t.Fatalf("expected non-auth error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "garmin connectapi error") {
		t.Fatalf("unexpected error: %v", err)
	}
}
