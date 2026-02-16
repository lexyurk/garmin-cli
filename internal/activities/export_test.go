package activities

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lexyurk/garmin-cli/internal/auth"
	"github.com/lexyurk/garmin-cli/internal/client"
)

func TestExport_DownloadsToWriter(t *testing.T) {
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/download-service/export/gpx/activity/123" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Accept"); got != "*/*" {
			t.Fatalf("unexpected Accept header: %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("file-contents"))
	}))
	defer srv.Close()

	c := client.NewWithSession("ignored", "default", sess, client.Options{
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
	})

	var b bytes.Buffer
	if err := Export(context.Background(), c, 123, ExportGPX, &b); err != nil {
		t.Fatalf("Export error: %v", err)
	}
	if b.String() != "file-contents" {
		t.Fatalf("unexpected body: %q", b.String())
	}
}

func TestExport_Maps401ToErrNotAuthenticated(t *testing.T) {
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("nope"))
	}))
	defer srv.Close()

	c := client.NewWithSession("ignored", "default", sess, client.Options{
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
	})

	var b bytes.Buffer
	err := Export(context.Background(), c, 123, ExportGPX, &b)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, auth.ErrNotAuthenticated) {
		t.Fatalf("expected ErrNotAuthenticated, got: %v", err)
	}
}
