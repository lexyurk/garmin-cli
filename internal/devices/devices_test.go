package devices

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

func TestList_ParsesDevices(t *testing.T) {
	c := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/device-service/deviceregistration/devices" {
			t.Fatalf("path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
		  {"deviceId":111,"productDisplayName":"Forerunner 965","serialNumber":"abc","partNumber":"006-X"},
		  {"deviceId":222,"displayName":"HRM-Pro"}
		]`))
	})

	list, err := List(context.Background(), c)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(list))
	}
	if list[0].DeviceID != 111 || list[0].Name != "Forerunner 965" {
		t.Fatalf("device0: %#v", list[0])
	}
	// falls back to displayName when productDisplayName is absent
	if list[1].Name != "HRM-Pro" {
		t.Fatalf("device1 name fallback: %#v", list[1])
	}
}
