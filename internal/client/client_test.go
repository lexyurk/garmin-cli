package client

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lexyurk/garmin-cli/internal/auth"
)

func TestClient_RefreshesExpiredOAuth2(t *testing.T) {
	now := time.Now()
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "old", ExpiresAt: now.Add(-time.Minute).Unix()},
	}

	refreshed := false
	saved := false

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer new" {
			t.Fatalf("unexpected Authorization header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewWithSession("ignored", "default", sess, Options{
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
		RefreshOAuth2: func(ctx context.Context, configDir string, oauth1 auth.OAuth1Token) (auth.OAuth2Token, error) {
			refreshed = true
			return auth.OAuth2Token{TokenType: "bearer", AccessToken: "new", ExpiresAt: time.Now().Add(time.Hour).Unix()}, nil
		},
		SaveSession: func(configDir, profile string, s *auth.Session) error {
			saved = true
			return nil
		},
	})

	var out map[string]any
	if err := c.GetJSON(context.Background(), "/ping", nil, &out); err != nil {
		t.Fatalf("GetJSON error: %v", err)
	}

	if refreshed != true {
		t.Fatalf("expected refresh to be called")
	}
	if saved != true {
		t.Fatalf("expected session to be saved after refresh")
	}
}

func TestClient_DoesNotRefreshWhenNotExpired(t *testing.T) {
	now := time.Now()
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: now.Add(time.Hour).Unix()},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer ok" {
			t.Fatalf("unexpected Authorization header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewWithSession("ignored", "default", sess, Options{
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
		RefreshOAuth2: func(ctx context.Context, configDir string, oauth1 auth.OAuth1Token) (auth.OAuth2Token, error) {
			t.Fatalf("refresh should not be called")
			return auth.OAuth2Token{}, nil
		},
		SaveSession: func(configDir, profile string, s *auth.Session) error {
			t.Fatalf("save should not be called")
			return nil
		},
	})

	var out map[string]any
	if err := c.GetJSON(context.Background(), "/ping", nil, &out); err != nil {
		t.Fatalf("GetJSON error: %v", err)
	}
}

func TestClient_Maps401ToNotAuthenticated(t *testing.T) {
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"nope"}`))
	}))
	defer srv.Close()

	c := NewWithSession("ignored", "default", sess, Options{
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
	})

	var out map[string]any
	err := c.GetJSON(context.Background(), "/ping", nil, &out)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, auth.ErrNotAuthenticated) {
		t.Fatalf("expected ErrNotAuthenticated, got: %v", err)
	}
}

func TestClient_RetriesOn429(t *testing.T) {
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}

	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"slow down"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewWithSession("ignored", "default", sess, Options{
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
	})

	var out map[string]any
	if err := c.GetJSON(context.Background(), "/ping", nil, &out); err != nil {
		t.Fatalf("GetJSON error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestClient_Logf_DoesNotLeakAuthorization(t *testing.T) {
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "super-secret-access-token", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}

	var logBuf bytes.Buffer

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewWithSession("ignored", "default", sess, Options{
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
		Logf: func(format string, args ...any) {
			_, _ = logBuf.WriteString("LOG: ")
			_, _ = fmt.Fprintf(&logBuf, format, args...)
			_, _ = logBuf.WriteString("\n")
		},
	})

	var out map[string]any
	if err := c.GetJSON(context.Background(), "/ping", nil, &out); err != nil {
		t.Fatalf("GetJSON error: %v", err)
	}

	logs := logBuf.String()
	if !strings.Contains(logs, "connectapi:") {
		t.Fatalf("expected logs to contain connectapi prefix, got:\n%s", logs)
	}
	if strings.Contains(logs, "Authorization") || strings.Contains(logs, "super-secret-access-token") {
		t.Fatalf("expected logs not to leak auth, got:\n%s", logs)
	}
}

func TestClient_RetriesOn429_HonorsRetryAfter(t *testing.T) {
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}

	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "2")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"message":"slow down"}`))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	var slept time.Duration
	c := NewWithSession("ignored", "default", sess, Options{
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
		Sleep: func(d time.Duration) {
			slept = d
		},
	})

	var out map[string]any
	if err := c.GetJSON(context.Background(), "/ping", nil, &out); err != nil {
		t.Fatalf("GetJSON error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if slept < 2*time.Second {
		t.Fatalf("expected sleep to honor Retry-After >=2s, got %s", slept)
	}
}
