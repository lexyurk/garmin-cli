package records

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

func TestList_LabelsKnownTypes(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/personalrecord-service/personalrecord/prs/runner" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
		  {"typeId":3,"value":1350,"activityId":7,"activityName":"5k TT"},
		  {"typeId":99,"value":42195}
		]`))
	})

	out, err := List(context.Background(), c, "runner")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2, got %d", len(out))
	}
	if out[0].Label != "5 km" || out[0].ActivityID != 7 {
		t.Fatalf("record0: %#v", out[0])
	}
	if out[1].Label != "type 99" {
		t.Fatalf("unknown type should fall back: %q", out[1].Label)
	}
}

func TestList_RequiresDisplayName(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("should not call API")
	})
	if _, err := List(context.Background(), c, ""); err == nil {
		t.Fatalf("expected error for empty display name")
	}
}

func TestListRaw_PassesThrough(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/personalrecord-service/personalrecord/prs/runner" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"typeId":3,"value":1350}]`))
	})
	raw, err := ListRaw(context.Background(), c, "runner")
	if err != nil {
		t.Fatalf("ListRaw: %v", err)
	}
	if len(raw) != 1 || raw[0]["typeId"].(float64) != 3 {
		t.Fatalf("unexpected: %#v", raw)
	}
	if _, err := ListRaw(context.Background(), c, "  "); err == nil {
		t.Fatalf("expected error for empty display name")
	}
}
