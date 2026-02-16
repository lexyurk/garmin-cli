package activities

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lexyurk/garmin-cli/internal/auth"
	"github.com/lexyurk/garmin-cli/internal/client"
)

func TestList_ValidatesDateFlags(t *testing.T) {
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}
	// Should never be called for validation failures.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("unexpected request: %s", r.URL.String())
	}))
	defer srv.Close()

	c := client.NewWithSession("ignored", "default", sess, client.Options{
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
	})

	_, err := List(context.Background(), c, 10, "not-a-date", "", "")
	if err == nil {
		t.Fatalf("expected error")
	}

	_, err = List(context.Background(), c, 10, "2026-02-10", "2026-02-01", "")
	if err == nil {
		t.Fatalf("expected error")
	}
}

