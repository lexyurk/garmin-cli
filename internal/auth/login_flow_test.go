package auth

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/lexyurk/garmin-cli/internal/config"
)

type flowState struct {
	mfaCode        string
	exchangeMFATok string
}

func newAuthFlowServer(t *testing.T, requireMFA bool) (*httptest.Server, *flowState) {
	t.Helper()
	state := &flowState{}

	const (
		csrf1  = "csrf-1"
		csrf2  = "csrf-2"
		ticket = "TICKET123"
	)

	signinHTML := func(csrf, title string) string {
		return `<html><head><title>` + title + `</title></head><body>` +
			`<input type="hidden" name="_csrf" value="` + csrf + `"/>` +
			`</body></html>`
	}
	ticketHTML := func() string {
		// Regex expects embed?ticket=... followed by a quote.
		return `<html><head><title>OK</title></head><body>` +
			`<a href="https://sso.garmin.com/sso/embed?ticket=` + ticket + `">continue</a>` +
			`</body></html>`
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sso/embed":
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "<html>embed</html>")
			return

		case "/sso/signin":
			switch r.Method {
			case http.MethodGet:
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, signinHTML(csrf1, "Sign In"))
				return
			case http.MethodPost:
				body, _ := io.ReadAll(r.Body)
				vals, _ := url.ParseQuery(string(body))
				if vals.Get("username") == "" || vals.Get("password") == "" {
					t.Fatalf("expected username/password form fields, got: %q", string(body))
				}
				if vals.Get("_csrf") != csrf1 {
					t.Fatalf("expected csrf=%q, got %q", csrf1, vals.Get("_csrf"))
				}

				w.WriteHeader(http.StatusOK)
				if requireMFA {
					_, _ = io.WriteString(w, signinHTML(csrf2, "MFA"))
				} else {
					_, _ = io.WriteString(w, ticketHTML())
				}
				return
			default:
				t.Fatalf("unexpected method: %s", r.Method)
			}

		case "/sso/verifyMFA/loginEnterMfaCode":
			body, _ := io.ReadAll(r.Body)
			vals, _ := url.ParseQuery(string(body))
			state.mfaCode = vals.Get("mfa-code")
			if vals.Get("_csrf") != csrf2 {
				t.Fatalf("expected csrf=%q, got %q", csrf2, vals.Get("_csrf"))
			}
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, ticketHTML())
			return

		case "/oauth_consumer.json":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{"consumer_key":"k","consumer_secret":"s"}`)
			return

		case "/oauth-service/oauth/preauthorized":
			if got := strings.TrimSpace(r.URL.Query().Get("ticket")); got != ticket {
				t.Fatalf("unexpected ticket query: %q", got)
			}
			w.WriteHeader(http.StatusOK)
			if requireMFA {
				_, _ = io.WriteString(w, "oauth_token=o1&oauth_token_secret=o1s&mfa_token=mfa-token-xyz")
			} else {
				_, _ = io.WriteString(w, "oauth_token=o1&oauth_token_secret=o1s")
			}
			return

		case "/oauth-service/oauth/exchange/user/2.0":
			body, _ := io.ReadAll(r.Body)
			vals, _ := url.ParseQuery(string(body))
			state.exchangeMFATok = vals.Get("mfa_token")

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, `{
			  "token_type":"bearer",
			  "access_token":"at",
			  "refresh_token":"rt",
			  "expires_in":3600,
			  "refresh_token_expires_in":7200
			}`)
			return
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))

	return srv, state
}

func withRewriteTransport(t *testing.T, srv *httptest.Server) {
	t.Helper()
	target, _ := url.Parse(srv.URL)

	orig := defaultTransport
	defaultTransport = roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		r2 := r.Clone(r.Context())
		r2.URL.Scheme = target.Scheme
		r2.URL.Host = target.Host
		return http.DefaultTransport.RoundTrip(r2)
	})
	t.Cleanup(func() { defaultTransport = orig })
}

func TestLogin_HappyPath_NoMFA(t *testing.T) {
	srv, _ := newAuthFlowServer(t, false)
	defer srv.Close()
	withRewriteTransport(t, srv)

	dir := t.TempDir()
	sess, err := Login(context.Background(), dir, "user@example.com", "pass", nil)
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}
	if sess.OAuth1.OAuthToken != "o1" || sess.OAuth2.AccessToken != "at" {
		t.Fatalf("unexpected session: %#v", sess)
	}
	if sess.OAuth2.ExpiresAt == 0 || sess.OAuth2.RefreshTokenExpiresAt == 0 {
		t.Fatalf("expected expiry timestamps to be set: %#v", sess.OAuth2)
	}

	// Consumer cache should be written as a side effect.
	if _, err := os.Stat(config.OAuthConsumerCachePath(dir)); err != nil {
		t.Fatalf("expected oauth consumer cache file: %v", err)
	}
}

func TestLogin_HappyPath_MFA(t *testing.T) {
	srv, st := newAuthFlowServer(t, true)
	defer srv.Close()
	withRewriteTransport(t, srv)

	dir := t.TempDir()
	sess, err := Login(context.Background(), dir, "user@example.com", "pass", func() (string, error) {
		return " 123456 \n", nil
	})
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}
	if st.mfaCode != "123456" {
		t.Fatalf("expected trimmed mfa code, got %q", st.mfaCode)
	}
	if sess.OAuth1.MFAToken != "mfa-token-xyz" {
		t.Fatalf("expected MFAToken propagated, got %#v", sess.OAuth1)
	}
	if st.exchangeMFATok != "mfa-token-xyz" {
		t.Fatalf("expected exchange mfa_token to be forwarded, got %q", st.exchangeMFATok)
	}
}

func TestLogin_MFARequired_WhenNoPrompt(t *testing.T) {
	srv, _ := newAuthFlowServer(t, true)
	defer srv.Close()
	withRewriteTransport(t, srv)

	_, err := Login(context.Background(), t.TempDir(), "user@example.com", "pass", nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !errors.Is(err, ErrMFARequired) {
		t.Fatalf("expected ErrMFARequired, got: %v", err)
	}
}

func TestRefreshOAuth2_ExchangesOAuth2(t *testing.T) {
	srv, st := newAuthFlowServer(t, false)
	defer srv.Close()
	withRewriteTransport(t, srv)

	dir := t.TempDir()
	tok, err := RefreshOAuth2(context.Background(), dir, OAuth1Token{OAuthToken: "o1", OAuthTokenSecret: "o1s", MFAToken: "mfa-token-abc"})
	if err != nil {
		t.Fatalf("RefreshOAuth2 error: %v", err)
	}
	if tok.AccessToken != "at" || tok.RefreshToken != "rt" {
		t.Fatalf("unexpected oauth2 token: %#v", tok)
	}
	if tok.ExpiresAt == 0 {
		t.Fatalf("expected ExpiresAt set")
	}
	// Server echoes mfa_token from request body; ensure exchangeOAuth2 includes it when present.
	if st.exchangeMFATok != "mfa-token-abc" {
		t.Fatalf("expected exchange mfa_token to be forwarded, got %q", st.exchangeMFATok)
	}
}
