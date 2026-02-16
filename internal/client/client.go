// Package client provides an HTTP client for the Garmin Connect API.
package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/lexyurk/garmin-cli/internal/auth"
)

const (
	baseURL = "https://connectapi.garmin.com"
)

// Client is an authenticated HTTP client for Garmin Connect.
type Client struct {
	httpClient *http.Client
	baseURL    string

	configDir string
	profile   string
	session   *auth.Session

	mu        sync.RWMutex
	refreshMu sync.Mutex

	refreshOAuth2 func(ctx context.Context, configDir string, oauth1 auth.OAuth1Token) (auth.OAuth2Token, error)
	saveSession   func(configDir, profile string, s *auth.Session) error

	logf func(format string, args ...any)
}

type Options struct {
	HTTPClient    *http.Client
	BaseURL       string
	RefreshOAuth2 func(ctx context.Context, configDir string, oauth1 auth.OAuth1Token) (auth.OAuth2Token, error)
	SaveSession   func(configDir, profile string, s *auth.Session) error
	Logf          func(format string, args ...any)
}

// New loads tokens for profile and returns a ready-to-use client.
func New(configDir, profile string, opts Options) (*Client, error) {
	sess, err := auth.LoadSession(configDir, profile)
	if err != nil {
		return nil, err
	}
	return NewWithSession(configDir, profile, sess, opts), nil
}

func NewWithSession(configDir, profile string, session *auth.Session, opts Options) *Client {
	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	u := opts.BaseURL
	if u == "" {
		u = baseURL
	}

	refreshFn := opts.RefreshOAuth2
	if refreshFn == nil {
		refreshFn = auth.RefreshOAuth2
	}
	saveFn := opts.SaveSession
	if saveFn == nil {
		saveFn = auth.SaveSession
	}

	return &Client{
		httpClient:    httpClient,
		baseURL:       u,
		configDir:     configDir,
		profile:       profile,
		session:       session,
		refreshOAuth2: refreshFn,
		saveSession:   saveFn,
		logf:          opts.Logf,
	}
}

func (c *Client) logfSafe(format string, args ...any) {
	if c.logf == nil {
		return
	}
	c.logf(format, args...)
}

// Do performs an authenticated request to the Connect API.
func (c *Client) Do(ctx context.Context, method, path string, query url.Values, body io.Reader, contentType string) (*http.Response, error) {
	if err := c.ensureFreshOAuth2(ctx); err != nil {
		return nil, err
	}

	u := c.baseURL + path
	if query != nil && len(query) > 0 {
		u += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "com.garmin.android.apps.connectmobile")

	c.mu.RLock()
	tokenType := ""
	accessToken := ""
	if c.session != nil {
		tokenType = c.session.OAuth2.TokenType
		accessToken = c.session.OAuth2.AccessToken
	}
	c.mu.RUnlock()
	req.Header.Set("Authorization", fmt.Sprintf("%s %s", stringsTitle(tokenType), accessToken))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")

	return c.doWithRetry(req)
}

func (c *Client) GetJSON(ctx context.Context, path string, query url.Values, out any) error {
	resp, err := c.Do(ctx, http.MethodGet, path, query, nil, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return fmt.Errorf("%w: %s: %s", auth.ErrNotAuthenticated, resp.Status, stringsTrim(string(b)))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return fmt.Errorf("garmin connectapi error: %s: %s", resp.Status, stringsTrim(string(b)))
	}

	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) ensureFreshOAuth2(ctx context.Context) error {
	c.mu.RLock()
	if c.session == nil {
		c.mu.RUnlock()
		return auth.ErrNotAuthenticated
	}
	expired := c.session.OAuth2.Expired(time.Now())
	oauth1 := c.session.OAuth1
	c.mu.RUnlock()
	if !expired {
		return nil
	}

	// Only one goroutine should refresh at a time. Others will re-check once it completes.
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()

	c.mu.RLock()
	if c.session == nil {
		c.mu.RUnlock()
		return auth.ErrNotAuthenticated
	}
	expired = c.session.OAuth2.Expired(time.Now())
	oauth1 = c.session.OAuth1
	c.mu.RUnlock()
	if !expired {
		return nil
	}

	c.logfSafe("connectapi: oauth2 expired; refreshing")
	oauth2, err := c.refreshOAuth2(ctx, c.configDir, oauth1)
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.session.OAuth2 = oauth2
	snapshot := *c.session
	c.mu.Unlock()

	if err := c.saveSession(c.configDir, c.profile, &snapshot); err != nil {
		return err
	}
	c.logfSafe("connectapi: oauth2 refreshed")
	return nil
}

func (c *Client) doWithRetry(req *http.Request) (*http.Response, error) {
	// Basic retry for rate limits and transient errors.
	const maxRetries = 3
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		r := req
		if attempt > 0 {
			if req.GetBody != nil {
				body, err := req.GetBody()
				if err != nil {
					return nil, err
				}
				r = req.Clone(req.Context())
				r.Body = body
			} else if req.Body != nil {
				// Can't safely retry requests with a non-rewindable body.
				return nil, lastErr
			}
		}

		attemptStart := time.Now()
		resp, err := c.httpClient.Do(r)
		attemptDur := time.Since(attemptStart)

		if err != nil {
			c.logfSafe("connectapi: %s %s attempt=%d error=%v", r.Method, r.URL.Path, attempt, err)
		} else if resp != nil {
			c.logfSafe("connectapi: %s %s attempt=%d status=%d dur=%s", r.Method, r.URL.Path, attempt, resp.StatusCode, attemptDur.Round(time.Millisecond))
		}
		if err == nil && resp != nil && resp.StatusCode < 500 && resp.StatusCode != 429 {
			return resp, nil
		}

		// If we got a response we won't return, close it before retrying.
		if resp != nil {
			io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))
			resp.Body.Close()
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("request failed: %s", resp.Status)
		}

		// Don't retry on the final attempt.
		if attempt == maxRetries {
			break
		}
		time.Sleep(time.Duration(200*(1<<attempt)) * time.Millisecond)
	}
	return nil, lastErr
}

func stringsTitle(s string) string {
	if s == "" {
		return "Bearer"
	}
	// Garmin returns "bearer"; normalize only first letter for readability.
	return strings.ToUpper(s[:1]) + s[1:]
}

func stringsTrim(s string) string {
	return strings.TrimSpace(s)
}
