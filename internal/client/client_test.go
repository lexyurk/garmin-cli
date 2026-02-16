package client

import (
	"context"
	"net/http"
	"net/http/httptest"
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

