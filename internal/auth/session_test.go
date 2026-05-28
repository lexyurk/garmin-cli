package auth

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lexyurk/garmin-cli/internal/config"
)

func TestSaveSessionAndLoadSession_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	want := &Session{
		OAuth1: OAuth1Token{
			OAuthToken:       "o1",
			OAuthTokenSecret: "o1s",
			Domain:           "garmin.com",
		},
		OAuth2: OAuth2Token{
			TokenType:    "bearer",
			AccessToken:  "at",
			RefreshToken: "rt",
			ExpiresAt:    time.Now().Add(time.Hour).Unix(),
		},
	}

	if err := SaveSession(dir, "work", want); err != nil {
		t.Fatalf("SaveSession error: %v", err)
	}

	o1Path := config.OAuth1TokenPath(dir, "work")
	o2Path := config.OAuth2TokenPath(dir, "work")

	for _, p := range []string{o1Path, o2Path} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatalf("stat %s: %v", p, err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("unexpected perms for %s: %v", p, info.Mode().Perm())
		}
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		if len(b) == 0 || b[len(b)-1] != '\n' {
			t.Fatalf("expected trailing newline in %s", p)
		}
	}

	got, err := LoadSession(dir, "work")
	if err != nil {
		t.Fatalf("LoadSession error: %v", err)
	}
	if got.OAuth1.OAuthToken != want.OAuth1.OAuthToken || got.OAuth2.AccessToken != want.OAuth2.AccessToken {
		t.Fatalf("unexpected loaded session: %#v", got)
	}
}

func TestLoadSession_MissingTokensReturnsErrNotAuthenticated(t *testing.T) {
	dir := t.TempDir()

	_, err := LoadSession(dir, "default")
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("expected ErrNotAuthenticated, got: %v", err)
	}
}

func TestSaveSession_InvalidProfileRejected(t *testing.T) {
	if err := SaveSession(t.TempDir(), "..", &Session{}); err == nil {
		t.Fatalf("expected error")
	}
}

func TestLoadJSON_WrapsParseErrors(t *testing.T) {
	dir := t.TempDir()
	path := config.OAuth1TokenPath(dir, "default")
	if err := os.MkdirAll(config.TokensDir(dir, "default"), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{not valid json"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := loadJSON[OAuth1Token](path)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "parse "+path) {
		t.Fatalf("expected wrapped parse error to mention path, got: %v", err)
	}
}
