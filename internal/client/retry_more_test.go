package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lexyurk/garmin-cli/internal/auth"
)

func TestRetryAfterDelay_SecondsAndDate(t *testing.T) {
	now := time.Unix(1_700_000_000, 0).UTC()

	if got := retryAfterDelay("", now); got != 0 {
		t.Fatalf("expected 0, got %s", got)
	}
	if got := retryAfterDelay("0", now); got != 0 {
		t.Fatalf("expected 0, got %s", got)
	}
	if got := retryAfterDelay("-1", now); got != 0 {
		t.Fatalf("expected 0, got %s", got)
	}
	if got := retryAfterDelay("2", now); got != 2*time.Second {
		t.Fatalf("expected 2s, got %s", got)
	}

	future := now.Add(5 * time.Second).Format(http.TimeFormat)
	if got := retryAfterDelay(future, now); got != 5*time.Second {
		t.Fatalf("expected 5s, got %s", got)
	}
	if got := retryAfterDelay("not-a-date", now); got != 0 {
		t.Fatalf("expected 0, got %s", got)
	}
}

func TestStringsTitle(t *testing.T) {
	if got := stringsTitle(""); got != "Bearer" {
		t.Fatalf("expected Bearer, got %q", got)
	}
	if got := stringsTitle("bearer"); got != "Bearer" {
		t.Fatalf("expected Bearer, got %q", got)
	}
	if got := stringsTitle("Token"); got != "Token" {
		t.Fatalf("expected Token, got %q", got)
	}
}

func TestOAuth2ExpiresAt_NilSessionIsZero(t *testing.T) {
	c := NewWithSession("ignored", "default", nil, Options{})
	if got := c.OAuth2ExpiresAt(); got != 0 {
		t.Fatalf("expected 0, got %d", got)
	}
}

func TestDoWithRetry_NonRewindableBodyDoesNotRetry(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("nope"))
	}))
	defer srv.Close()

	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}
	c := NewWithSession("ignored", "default", sess, Options{
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
		Sleep: func(d time.Duration) {
			// Don't actually sleep in tests.
		},
	})

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, srv.URL+"/x", io.NopCloser(strings.NewReader("body")))
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	// Ensure GetBody is nil (io.NopCloser isn't one of the rewindable types handled by net/http).
	req.GetBody = nil

	resp, err := c.doWithRetry(req)
	if err == nil || resp != nil {
		t.Fatalf("expected error and nil response, got resp=%v err=%v", resp, err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 attempt due to non-rewindable body, got %d", calls)
	}
}
