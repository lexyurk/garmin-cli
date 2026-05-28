package activities

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

func manageTestClient(t *testing.T, handler http.HandlerFunc) *client.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}
	return client.NewWithSession("ignored", "default", sess, client.Options{HTTPClient: srv.Client(), BaseURL: srv.URL})
}

func TestResolveType(t *testing.T) {
	types := []ActivityType{
		{TypeID: 1, TypeKey: "running", ParentTypeID: 17},
		{TypeID: 2, TypeKey: "cycling", ParentTypeID: 17},
	}
	got, err := ResolveType(types, "Running")
	if err != nil {
		t.Fatalf("ResolveType: %v", err)
	}
	if got.TypeID != 1 {
		t.Fatalf("typeId: %d", got.TypeID)
	}
	if _, err := ResolveType(types, "swimming"); err == nil {
		t.Fatalf("expected error for unknown type")
	}
}

func TestUpdate_SendsPartialBody(t *testing.T) {
	var method, path string
	var body map[string]any
	c := manageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		_ = json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(http.StatusNoContent)
	})

	name := "Morning Run"
	if err := Update(context.Background(), c, 42, UpdateOptions{Name: &name}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if method != http.MethodPut || path != "/activity-service/activity/42" {
		t.Fatalf("unexpected request: %s %s", method, path)
	}
	if body["activityName"] != "Morning Run" {
		t.Fatalf("activityName missing: %#v", body)
	}
	if _, ok := body["description"]; ok {
		t.Fatalf("description should be omitted: %#v", body)
	}
}

func TestUpdate_NothingToUpdate(t *testing.T) {
	c := manageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not call API")
	})
	if err := Update(context.Background(), c, 42, UpdateOptions{}); err == nil {
		t.Fatalf("expected error when no fields set")
	}
}

func TestDelete_SendsDelete(t *testing.T) {
	var method, path string
	c := manageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	})
	if err := Delete(context.Background(), c, 42); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if method != http.MethodDelete || path != "/activity-service/activity/42" {
		t.Fatalf("unexpected request: %s %s", method, path)
	}
}

func TestListByGear_Path(t *testing.T) {
	c := manageTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activitylist-service/activities/abc-uuid/gear" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"activityId":7,"activityName":"Run","activityType":{"typeKey":"running"},"distance":5000,"duration":1500}]`))
	})
	out, err := ListByGear(context.Background(), c, "abc-uuid", 10)
	if err != nil {
		t.Fatalf("ListByGear: %v", err)
	}
	if len(out) != 1 || out[0].ID != 7 || out[0].Type != "running" {
		t.Fatalf("unexpected: %#v", out)
	}
}
