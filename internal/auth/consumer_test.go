package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/lexyurk/garmin-cli/internal/config"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestLoadCachedConsumer_TTLAndValidation(t *testing.T) {
	dir := t.TempDir()
	path := config.OAuthConsumerCachePath(dir)

	okCache := oauthConsumerCache{ConsumerKey: "k", ConsumerSecret: "s", FetchedAt: time.Now().Unix()}
	if err := saveJSON(path, okCache, 0o600); err != nil {
		t.Fatalf("saveJSON: %v", err)
	}
	c, ok := loadCachedConsumer(path)
	if !ok {
		t.Fatalf("expected cache hit")
	}
	if c.ConsumerKey != "k" || c.ConsumerSecret != "s" {
		t.Fatalf("unexpected consumer: %#v", c)
	}

	expired := oauthConsumerCache{ConsumerKey: "k", ConsumerSecret: "s", FetchedAt: time.Now().Add(-consumerCacheTTL - time.Hour).Unix()}
	if err := saveJSON(path, expired, 0o600); err != nil {
		t.Fatalf("saveJSON: %v", err)
	}
	if _, ok := loadCachedConsumer(path); ok {
		t.Fatalf("expected expired cache to be ignored")
	}

	future := oauthConsumerCache{ConsumerKey: "k", ConsumerSecret: "s", FetchedAt: time.Now().Add(time.Hour).Unix()}
	if err := saveJSON(path, future, 0o600); err != nil {
		t.Fatalf("saveJSON: %v", err)
	}
	if _, ok := loadCachedConsumer(path); ok {
		t.Fatalf("expected future cache to be ignored")
	}

	invalid := oauthConsumerCache{ConsumerKey: "", ConsumerSecret: "s", FetchedAt: time.Now().Unix()}
	if err := saveJSON(path, invalid, 0o600); err != nil {
		t.Fatalf("saveJSON: %v", err)
	}
	if _, ok := loadCachedConsumer(path); ok {
		t.Fatalf("expected invalid cache to be ignored")
	}
}

func TestGetOAuthConsumer_UsesCacheWithoutNetwork(t *testing.T) {
	dir := t.TempDir()
	path := config.OAuthConsumerCachePath(dir)
	if err := os.MkdirAll(config.CacheDir(dir), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	okCache := oauthConsumerCache{ConsumerKey: "k", ConsumerSecret: "s", FetchedAt: time.Now().Unix()}
	if err := saveJSON(path, okCache, 0o600); err != nil {
		t.Fatalf("saveJSON: %v", err)
	}

	orig := defaultTransport
	defaultTransport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		t.Fatalf("unexpected network request: %s", r.URL.String())
		return nil, nil
	})
	t.Cleanup(func() { defaultTransport = orig })

	c, err := getOAuthConsumer(context.Background(), dir, nil)
	if err != nil {
		t.Fatalf("getOAuthConsumer error: %v", err)
	}
	if c.ConsumerKey != "k" || c.ConsumerSecret != "s" {
		t.Fatalf("unexpected consumer: %#v", c)
	}
}

func TestGetOAuthConsumer_FetchesAndCaches(t *testing.T) {
	dir := t.TempDir()

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/oauth_consumer.json" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"consumer_key":"k","consumer_secret":"s"}`))
	}))
	defer srv.Close()

	target, _ := url.Parse(srv.URL)
	orig := defaultTransport
	defaultTransport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		r2 := r.Clone(r.Context())
		r2.URL.Scheme = target.Scheme
		r2.URL.Host = target.Host
		return http.DefaultTransport.RoundTrip(r2)
	})
	t.Cleanup(func() { defaultTransport = orig })

	c, err := getOAuthConsumer(context.Background(), dir, nil)
	if err != nil {
		t.Fatalf("getOAuthConsumer error: %v", err)
	}
	if c.ConsumerKey != "k" || c.ConsumerSecret != "s" {
		t.Fatalf("unexpected consumer: %#v", c)
	}
	if calls != 1 {
		t.Fatalf("expected 1 fetch call, got %d", calls)
	}

	// Second call should hit cache (no extra HTTP calls).
	c2, err := getOAuthConsumer(context.Background(), dir, nil)
	if err != nil {
		t.Fatalf("getOAuthConsumer error: %v", err)
	}
	if c2.ConsumerKey != "k" || calls != 1 {
		t.Fatalf("expected cached consumer, calls=%d consumer=%#v", calls, c2)
	}
}
