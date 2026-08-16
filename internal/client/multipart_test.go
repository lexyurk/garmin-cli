package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lexyurk/garmin-cli/internal/auth"
)

func TestPostMultipartFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s", r.Method)
		}
		if err := r.ParseMultipartForm(1024); err != nil {
			t.Fatal(err)
		}
		file, header, err := r.FormFile("fi_eld")
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		if header.Filename != "route_name.gpx" || header.Header.Get("Content-Type") != "application/gpx+xml" || string(data) != "gpx" {
			t.Fatalf("file=%q type=%q data=%q", header.Filename, header.Header.Get("Content-Type"), data)
		}
		_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	}))
	defer srv.Close()
	sess := &auth.Session{OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "x", ExpiresAt: time.Now().Add(time.Hour).Unix()}}
	c := NewWithSession("", "", sess, Options{HTTPClient: srv.Client(), BaseURL: srv.URL})
	var out map[string]bool
	if err := c.PostMultipartFile(context.Background(), "/upload", "fi\neld", "route\"name.gpx", "application/gpx+xml", []byte("gpx"), &out); err != nil {
		t.Fatal(err)
	}
	if !out["ok"] {
		t.Fatalf("out=%v", out)
	}
}

func TestPostMultipartFileStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "bad", http.StatusBadRequest) }))
	defer srv.Close()
	sess := &auth.Session{OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "x", ExpiresAt: time.Now().Add(time.Hour).Unix()}}
	c := NewWithSession("", "", sess, Options{HTTPClient: srv.Client(), BaseURL: srv.URL})
	if err := c.PostMultipartFile(context.Background(), "/upload", "file", "x.gpx", "", nil, nil); err == nil {
		t.Fatal("expected status error")
	}
}
