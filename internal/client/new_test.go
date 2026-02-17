package client

import (
	"errors"
	"testing"
	"time"

	"github.com/lexyurk/garmin-cli/internal/auth"
)

func TestNew_LoadsSessionAndOAuth2ExpiresAt(t *testing.T) {
	cfgDir := t.TempDir()
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}
	if err := auth.SaveSession(cfgDir, "default", sess); err != nil {
		t.Fatalf("SaveSession: %v", err)
	}

	c, err := New(cfgDir, "default", Options{})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	if got := c.OAuth2ExpiresAt(); got != sess.OAuth2.ExpiresAt {
		t.Fatalf("OAuth2ExpiresAt=%d want %d", got, sess.OAuth2.ExpiresAt)
	}
}

func TestNew_MissingTokensReturnsNotAuthenticated(t *testing.T) {
	cfgDir := t.TempDir()
	_, err := New(cfgDir, "default", Options{})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, auth.ErrNotAuthenticated) {
		t.Fatalf("expected ErrNotAuthenticated, got: %v", err)
	}
}
