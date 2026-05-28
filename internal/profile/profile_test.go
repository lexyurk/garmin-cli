package profile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/lexyurk/garmin-cli/internal/auth"
	"github.com/lexyurk/garmin-cli/internal/client"
)

func testClient(t *testing.T, handler http.HandlerFunc) *client.Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	sess := &auth.Session{
		OAuth1: auth.OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s"},
		OAuth2: auth.OAuth2Token{TokenType: "bearer", AccessToken: "ok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}
	return client.NewWithSession("ignored", "default", sess, client.Options{HTTPClient: srv.Client(), BaseURL: srv.URL})
}

func TestGetProfile(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/userprofile-service/socialProfile" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"profileId":987654,"displayName":"runner","fullName":"Sam Run","userName":"sam","location":"Berlin"}`))
	})

	p, err := Get(context.Background(), c)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if p.ProfileID != 987654 {
		t.Fatalf("profile id: %d", p.ProfileID)
	}
	if p.DisplayName != "runner" || p.FullName != "Sam Run" {
		t.Fatalf("unexpected profile: %#v", p)
	}

	pk, err := UserProfilePK(context.Background(), c)
	if err != nil {
		t.Fatalf("UserProfilePK: %v", err)
	}
	if pk != 987654 {
		t.Fatalf("pk: %d", pk)
	}
}
