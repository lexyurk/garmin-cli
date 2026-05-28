package gear

import (
	"context"
	"encoding/json"
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

func TestCreate_SendsExpectedBody(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"uuid":"new1","displayName":"My Pegasus","gearTypeName":"Shoes","gearStatusName":"active","maximumMeters":800000}`))
	})

	g, err := Create(context.Background(), c, 987, CreateOptions{Name: "My Pegasus", Make: "Nike", MaxMeters: 800000})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if gotMethod != http.MethodPost || gotPath != "/gear-service/gear" {
		t.Fatalf("unexpected request: %s %s", gotMethod, gotPath)
	}
	if gotBody["customMakeModel"] != "My Pegasus" || gotBody["displayName"] != "My Pegasus" {
		t.Fatalf("name not in body: %#v", gotBody)
	}
	if gotBody["gearTypeName"] != "Shoes" {
		t.Fatalf("default type not Shoes: %#v", gotBody["gearTypeName"])
	}
	if gotBody["gearStatusName"] != "active" {
		t.Fatalf("status not active: %#v", gotBody["gearStatusName"])
	}
	if g.UUID != "new1" || g.Name != "My Pegasus" {
		t.Fatalf("unexpected created gear: %#v", g)
	}
}

func TestCreate_RequiresName(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not call API without a name")
	})
	if _, err := Create(context.Background(), c, 1, CreateOptions{}); err == nil {
		t.Fatalf("expected error for missing name")
	}
}

func TestSetStatus_FlipsStatusAndPuts(t *testing.T) {
	var putBody map[string]any
	var putPath string
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/gear-service/gear/filterGear":
			_, _ = w.Write([]byte(`[{"uuid":"u1","displayName":"Pegasus","gearStatusName":"active"}]`))
		case r.Method == http.MethodPut:
			putPath = r.URL.Path
			_ = json.NewDecoder(r.Body).Decode(&putBody)
			_, _ = w.Write([]byte(`{"uuid":"u1","displayName":"Pegasus","gearStatusName":"retired"}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	})

	g, err := SetStatus(context.Background(), c, 987, "u1", "retired")
	if err != nil {
		t.Fatalf("SetStatus: %v", err)
	}
	if putPath != "/gear-service/gear/u1" {
		t.Fatalf("put path: %s", putPath)
	}
	if putBody["gearStatusName"] != "retired" {
		t.Fatalf("status not flipped in body: %#v", putBody)
	}
	if g.Status != "retired" {
		t.Fatalf("returned status: %q", g.Status)
	}
}

func TestSetStatus_NotFound(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	})
	if _, err := SetStatus(context.Background(), c, 1, "missing", "retired"); err == nil {
		t.Fatalf("expected not found error")
	}
}
