package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dghubble/oauth1"
	"github.com/lexyurk/garmin-cli/internal/config"
)

type readErrorBody struct{}

func (readErrorBody) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (readErrorBody) Close() error             { return nil }

func responseTransport(status int, body io.ReadCloser) roundTripperFunc {
	return func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: status,
			Status:     http.StatusText(status),
			Header:     make(http.Header),
			Body:       body,
			Request:    r,
		}, nil
	}
}

func TestDoRequest_ErrorBranches(t *testing.T) {
	if _, _, err := doRequest(context.Background(), http.DefaultClient, http.MethodGet, "://bad", nil, "", nil); err == nil {
		t.Fatal("expected request construction error")
	}
	transportErr := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("transport failed")
	})}
	if _, _, err := doRequest(context.Background(), transportErr, http.MethodGet, "http://example.test", nil, "", nil); err == nil {
		t.Fatal("expected transport error")
	}
	badStatus := &http.Client{Transport: responseTransport(http.StatusBadRequest, io.NopCloser(strings.NewReader("bad")))}
	if _, _, err := doRequest(context.Background(), badStatus, http.MethodGet, "http://example.test", nil, "", nil); err == nil {
		t.Fatal("expected status error")
	}
	badBody := &http.Client{Transport: responseTransport(http.StatusOK, readErrorBody{})}
	if _, _, err := doRequest(context.Background(), badBody, http.MethodGet, "http://example.test", nil, "", nil); err == nil {
		t.Fatal("expected body read error")
	}
}

func TestSSOExtractors_RejectMalformedHTML(t *testing.T) {
	if _, err := extractCSRF("<html/>"); err == nil {
		t.Fatal("expected CSRF error")
	}
	if extractTitle("<html/>") != "" {
		t.Fatal("expected empty title")
	}
	if _, err := extractTicket("<html/>"); err == nil {
		t.Fatal("expected ticket error")
	}
}

func newFailingLoginServer(t *testing.T, stage string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fail := func(at string) bool {
			if stage == at {
				http.Error(w, "rejected", http.StatusBadRequest)
				return true
			}
			return false
		}
		switch r.URL.Path {
		case "/sso/embed":
			if !fail("embed") {
				_, _ = io.WriteString(w, "embed")
			}
		case "/sso/signin":
			if r.Method == http.MethodGet {
				if fail("signin-get") {
					return
				}
				if stage == "csrf" {
					_, _ = io.WriteString(w, "<html/>")
					return
				}
				_, _ = io.WriteString(w, `<input name="_csrf" value="csrf">`)
				return
			}
			if fail("signin-post") {
				return
			}
			if strings.HasPrefix(stage, "mfa-") {
				if stage == "mfa-csrf" {
					_, _ = io.WriteString(w, `<title>MFA</title>`)
				} else {
					_, _ = io.WriteString(w, `<title>MFA</title><input name="_csrf" value="csrf2">`)
				}
				return
			}
			if stage == "ticket" {
				_, _ = io.WriteString(w, `<title>OK</title>`)
				return
			}
			_, _ = io.WriteString(w, `<a href="embed?ticket=TICKET">ok</a>`)
		case "/sso/verifyMFA/loginEnterMfaCode":
			if !fail("mfa-verify") {
				_, _ = io.WriteString(w, `<a href="embed?ticket=TICKET">ok</a>`)
			}
		case "/oauth_consumer.json":
			if !fail("consumer") {
				_, _ = io.WriteString(w, `{"consumer_key":"k","consumer_secret":"s"}`)
			}
		case "/oauth-service/oauth/preauthorized":
			if !fail("oauth1") {
				_, _ = io.WriteString(w, `oauth_token=o1&oauth_token_secret=o1s`)
			}
		case "/oauth-service/oauth/exchange/user/2.0":
			if !fail("oauth2") {
				_, _ = io.WriteString(w, `{"token_type":"bearer","access_token":"at"}`)
			}
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
}

func TestLogin_StageFailures(t *testing.T) {
	for _, stage := range []string{"embed", "signin-get", "csrf", "signin-post", "ticket", "consumer", "oauth1", "oauth2", "mfa-csrf", "mfa-verify"} {
		t.Run(stage, func(t *testing.T) {
			srv := newFailingLoginServer(t, stage)
			defer srv.Close()
			withRewriteTransport(t, srv)
			_, _, err := login(context.Background(), t.TempDir(), "user@example.com", "password", func() (string, error) {
				return "123456", nil
			})
			if err == nil {
				t.Fatal("expected login error")
			}
		})
	}

	t.Run("mfa prompt", func(t *testing.T) {
		srv := newFailingLoginServer(t, "mfa-prompt")
		defer srv.Close()
		withRewriteTransport(t, srv)
		_, _, err := login(context.Background(), t.TempDir(), "user@example.com", "password", func() (string, error) {
			return "", errors.New("prompt failed")
		})
		if err == nil || !strings.Contains(err.Error(), "prompt failed") {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func oauthContextWithTransport(rt http.RoundTripper) context.Context {
	return context.WithValue(context.Background(), oauth1.HTTPClient, &http.Client{Transport: rt})
}

func TestOAuthExchange_ValidationTransportStatusAndDecodeErrors(t *testing.T) {
	consumer := oauthConsumer{ConsumerKey: "key", ConsumerSecret: "secret"}
	if _, err := getOAuth1Token(context.Background(), consumer, " "); err == nil {
		t.Fatal("expected empty ticket error")
	}
	if _, err := exchangeOAuth2(context.Background(), consumer, OAuth1Token{}); err == nil {
		t.Fatal("expected missing OAuth1 token error")
	}

	transportFailure := roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("transport failed")
	})
	if _, err := getOAuth1Token(oauthContextWithTransport(transportFailure), consumer, "ticket"); err == nil {
		t.Fatal("expected OAuth1 transport error")
	}
	validO1 := OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "secret"}
	if _, err := exchangeOAuth2(oauthContextWithTransport(transportFailure), consumer, validO1); err == nil {
		t.Fatal("expected OAuth2 transport error")
	}

	for _, tc := range []struct {
		name   string
		status int
		body   string
	}{
		{"status", http.StatusBadRequest, "bad"},
		{"missing fields", http.StatusOK, "oauth_token=only"},
	} {
		t.Run("oauth1 "+tc.name, func(t *testing.T) {
			rt := responseTransport(tc.status, io.NopCloser(strings.NewReader(tc.body)))
			if _, err := getOAuth1Token(oauthContextWithTransport(rt), consumer, "ticket"); err == nil {
				t.Fatal("expected OAuth1 error")
			}
		})
	}
	for _, tc := range []struct {
		name   string
		status int
		body   io.ReadCloser
	}{
		{"status", http.StatusBadRequest, io.NopCloser(strings.NewReader("bad"))},
		{"decode", http.StatusOK, io.NopCloser(strings.NewReader("not-json"))},
		{"missing fields", http.StatusOK, io.NopCloser(strings.NewReader(`{}`))},
	} {
		t.Run("oauth2 "+tc.name, func(t *testing.T) {
			rt := responseTransport(tc.status, tc.body)
			if _, err := exchangeOAuth2(oauthContextWithTransport(rt), consumer, validO1); err == nil {
				t.Fatal("expected OAuth2 error")
			}
		})
	}
	_ = oauth1HTTPClient(nil, consumer, nil)
}

func TestConsumerFetch_ErrorResponses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		client *http.Client
	}{
		{"transport", &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("failed") })}},
		{"status", &http.Client{Transport: responseTransport(http.StatusBadRequest, io.NopCloser(strings.NewReader("bad")))}},
		{"decode", &http.Client{Transport: responseTransport(http.StatusOK, io.NopCloser(strings.NewReader("bad json")))}},
		{"missing", &http.Client{Transport: responseTransport(http.StatusOK, io.NopCloser(strings.NewReader(`{}`)))}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := getOAuthConsumer(context.Background(), t.TempDir(), tc.client); err == nil {
				t.Fatal("expected consumer error")
			}
		})
	}
}

func TestSessionPersistence_ErrorBranches(t *testing.T) {
	if _, err := LoadSession(t.TempDir(), ".."); err == nil {
		t.Fatal("expected invalid profile error")
	}

	dir := t.TempDir()
	writeFile(t, config.OAuth1TokenPath(dir, "default"), "not-json")
	if _, err := LoadSession(dir, "default"); err == nil || errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("expected OAuth1 parse error, got %v", err)
	}

	dir = t.TempDir()
	writeFile(t, config.OAuth1TokenPath(dir, "default"), `{}`)
	if _, err := LoadSession(dir, "default"); !errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("expected missing OAuth2 auth error, got %v", err)
	}
	writeFile(t, config.OAuth2TokenPath(dir, "default"), "not-json")
	if _, err := LoadSession(dir, "default"); err == nil || errors.Is(err, ErrNotAuthenticated) {
		t.Fatalf("expected OAuth2 parse error, got %v", err)
	}

	blocked := filepath.Join(t.TempDir(), "config-file")
	if err := os.WriteFile(blocked, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := SaveSession(blocked, "default", &Session{}); err == nil {
		t.Fatal("expected OAuth1 save error")
	}

	dir = t.TempDir()
	o2 := config.OAuth2TokenPath(dir, "default")
	if err := os.MkdirAll(o2, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := SaveSession(dir, "default", &Session{}); err == nil {
		t.Fatal("expected OAuth2 save error")
	}
}

func TestSaveJSON_MarshalError(t *testing.T) {
	if err := saveJSON(filepath.Join(t.TempDir(), "x"), func() {}, 0o600); err == nil {
		t.Fatal("expected marshal error")
	}
}

func TestRefreshOAuth2_ConsumerError(t *testing.T) {
	original := defaultTransport
	defaultTransport = roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("consumer unavailable")
	})
	t.Cleanup(func() { defaultTransport = original })
	if _, err := RefreshOAuth2(context.Background(), t.TempDir(), OAuth1Token{}); err == nil {
		t.Fatal("expected refresh error")
	}
}
