package gear

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

func TestList_ParsesAndNames(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("userProfilePk"); got != "987" {
			t.Fatalf("userProfilePk: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
		  {"uuid":"u1","gearPk":1,"displayName":"Pegasus 40","gearTypeName":"Shoes","gearStatusName":"active","maximumMeters":800000},
		  {"uuid":"u2","gearPk":2,"gearMakeName":"Hoka","gearModelName":"Clifton","gearTypeName":"Shoes","gearStatusName":"retired"}
		]`))
	})

	gears, err := List(context.Background(), c, 987)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(gears) != 2 {
		t.Fatalf("expected 2 gears, got %d", len(gears))
	}
	if gears[0].Name != "Pegasus 40" {
		t.Fatalf("name[0]: %q", gears[0].Name)
	}
	if gears[1].Name != "Hoka Clifton" {
		t.Fatalf("name[1] should fall back to make+model: %q", gears[1].Name)
	}
}

func TestFilterByStatus(t *testing.T) {
	gears := []Gear{{Status: "active"}, {Status: "retired"}, {Status: "active"}}
	if got := len(FilterByStatus(gears, "active")); got != 2 {
		t.Fatalf("active: %d", got)
	}
	if got := len(FilterByStatus(gears, "retired")); got != 1 {
		t.Fatalf("retired: %d", got)
	}
	if got := len(FilterByStatus(gears, "all")); got != 3 {
		t.Fatalf("all: %d", got)
	}
	if got := len(FilterByStatus(gears, "")); got != 3 {
		t.Fatalf("empty: %d", got)
	}
}

func TestGetStats(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gear-service/gear/stats/u1" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"totalDistance":123456.0,"totalActivities":42}`))
	})

	st, err := GetStats(context.Background(), c, "u1")
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if st.TotalMeters != 123456.0 || st.TotalActivities != 42 {
		t.Fatalf("unexpected stats: %#v", st)
	}
}

func TestGet_FindsByUUIDWithStats(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/gear-service/gear/filterGear":
			_, _ = w.Write([]byte(`[{"uuid":"u1","displayName":"Pegasus","gearStatusName":"active"}]`))
		case "/gear-service/gear/stats/u1":
			_, _ = w.Write([]byte(`{"totalDistance":5000,"totalActivities":3}`))
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	})

	g, err := Get(context.Background(), c, 1, "u1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if g.TotalMeters == nil || *g.TotalMeters != 5000 {
		t.Fatalf("total meters: %#v", g.TotalMeters)
	}
	if g.Activities == nil || *g.Activities != 3 {
		t.Fatalf("activities: %#v", g.Activities)
	}
}
