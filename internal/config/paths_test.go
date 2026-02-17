package config

import (
	"path/filepath"
	"testing"
)

func TestPathHelpers(t *testing.T) {
	dir := "/tmp/garmin-config"

	if got := TokensDir(dir, ""); got != filepath.Join(dir, "tokens", "default") {
		t.Fatalf("TokensDir default = %q", got)
	}
	if got := TokensDir(dir, "../work"); got != filepath.Join(dir, "tokens", "work") {
		t.Fatalf("TokensDir sanitizes = %q", got)
	}

	if got := CacheDir(dir); got != filepath.Join(dir, "cache") {
		t.Fatalf("CacheDir = %q", got)
	}
	if got := OAuthConsumerCachePath(dir); got != filepath.Join(dir, "cache", "oauth_consumer.json") {
		t.Fatalf("OAuthConsumerCachePath = %q", got)
	}

	if got := OAuth1TokenPath(dir, "work"); got != filepath.Join(dir, "tokens", "work", "oauth1_token.json") {
		t.Fatalf("OAuth1TokenPath = %q", got)
	}
	if got := OAuth2TokenPath(dir, "work"); got != filepath.Join(dir, "tokens", "work", "oauth2_token.json") {
		t.Fatalf("OAuth2TokenPath = %q", got)
	}
}
