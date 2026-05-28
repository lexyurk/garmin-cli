package calendar

import (
	"context"
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

func TestMonth_UsesZeroBasedMonth(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		// June (6) -> 0-based 5
		if r.URL.Path != "/calendar-service/year/2026/month/5" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"calendarItems":[
		  {"date":"2026-06-03","itemType":"workout","title":"Intervals","workoutId":111},
		  {"calendarDate":"2026-06-05","itemType":"activity","title":"Run","activityId":222}
		]}`))
	})

	items, err := Month(context.Background(), c, 2026, 6)
	if err != nil {
		t.Fatalf("Month: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].WorkoutID != 111 || items[0].Date != "2026-06-03" {
		t.Fatalf("item0: %#v", items[0])
	}
	// falls back to calendarDate when date is empty
	if items[1].Date != "2026-06-05" || items[1].ActivityID != 222 {
		t.Fatalf("item1: %#v", items[1])
	}
}

func TestMonth_RejectsBadMonth(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not call API")
	})
	if _, err := Month(context.Background(), c, 2026, 13); err == nil {
		t.Fatalf("expected error for month 13")
	}
}

func TestFilterByType(t *testing.T) {
	items := []Item{{Type: "workout"}, {Type: "activity"}, {Type: "workout"}}
	if got := len(FilterByType(items, "workout")); got != 2 {
		t.Fatalf("workout: %d", got)
	}
	if got := len(FilterByType(items, "")); got != 3 {
		t.Fatalf("all: %d", got)
	}
}
