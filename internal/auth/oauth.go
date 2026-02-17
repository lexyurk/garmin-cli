package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dghubble/oauth1"
)

func getOAuth1Token(ctx context.Context, consumer oauthConsumer, ticket string) (OAuth1Token, error) {
	if strings.TrimSpace(ticket) == "" {
		return OAuth1Token{}, errors.New("empty ticket")
	}

	loginURL := "https://sso.garmin.com/sso/embed"
	u, err := url.Parse("https://connectapi.garmin.com/oauth-service/oauth/preauthorized")
	if err != nil {
		return OAuth1Token{}, err
	}
	q := u.Query()
	q.Set("ticket", ticket)
	q.Set("login-url", loginURL)
	q.Set("accepts-mfa-tokens", "true")
	u.RawQuery = q.Encode()

	httpClient := oauth1HTTPClient(ctx, consumer, nil)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return OAuth1Token{}, err
	}
	req.Header.Set("User-Agent", "com.garmin.android.apps.connectmobile")

	resp, err := httpClient.Do(req)
	if err != nil {
		return OAuth1Token{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return OAuth1Token{}, fmt.Errorf("oauth1 preauthorized failed: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return OAuth1Token{}, err
	}

	values, err := url.ParseQuery(string(raw))
	if err != nil {
		return OAuth1Token{}, err
	}

	token := OAuth1Token{
		OAuthToken:       values.Get("oauth_token"),
		OAuthTokenSecret: values.Get("oauth_token_secret"),
		MFAToken:         values.Get("mfa_token"),
		Domain:           "garmin.com",
	}
	if token.OAuthToken == "" || token.OAuthTokenSecret == "" {
		return OAuth1Token{}, errors.New("oauth1 preauthorized response missing oauth_token fields")
	}
	return token, nil
}

func exchangeOAuth2(ctx context.Context, consumer oauthConsumer, oauth1Token OAuth1Token) (OAuth2Token, error) {
	if oauth1Token.OAuthToken == "" || oauth1Token.OAuthTokenSecret == "" {
		return OAuth2Token{}, errors.New("missing oauth1 token/secret")
	}

	u := "https://connectapi.garmin.com/oauth-service/oauth/exchange/user/2.0"

	form := url.Values{}
	if oauth1Token.MFAToken != "" {
		form.Set("mfa_token", oauth1Token.MFAToken)
	}
	body := strings.NewReader(form.Encode())

	httpClient := oauth1HTTPClient(ctx, consumer, &oauth1.Token{
		Token:       oauth1Token.OAuthToken,
		TokenSecret: oauth1Token.OAuthTokenSecret,
	})

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, body)
	if err != nil {
		return OAuth2Token{}, err
	}
	req.Header.Set("User-Agent", "com.garmin.android.apps.connectmobile")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := httpClient.Do(req)
	if err != nil {
		return OAuth2Token{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return OAuth2Token{}, fmt.Errorf("oauth2 exchange failed: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}

	var token OAuth2Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return OAuth2Token{}, err
	}

	now := time.Now()
	if token.ExpiresIn > 0 && token.ExpiresAt == 0 {
		token.ExpiresAt = now.Unix() + int64(token.ExpiresIn)
	}
	if token.RefreshTokenExpiresIn > 0 && token.RefreshTokenExpiresAt == 0 {
		token.RefreshTokenExpiresAt = now.Unix() + int64(token.RefreshTokenExpiresIn)
	}

	if token.TokenType == "" || token.AccessToken == "" {
		return OAuth2Token{}, errors.New("oauth2 exchange response missing token_type/access_token")
	}

	return token, nil
}

func oauth1HTTPClient(ctx context.Context, consumer oauthConsumer, token *oauth1.Token) *http.Client {
	cfg := oauth1.Config{
		ConsumerKey:    consumer.ConsumerKey,
		ConsumerSecret: consumer.ConsumerSecret,
	}
	if token == nil {
		token = &oauth1.Token{}
	}
	// oauth1.Transport uses http.DefaultTransport if its Base is nil. Provide a Base transport
	// via context so tests (and callers) can stub network without touching globals.
	if ctx == nil {
		ctx = context.Background()
	}
	if ctx.Value(oauth1.HTTPClient) == nil {
		ctx = context.WithValue(ctx, oauth1.HTTPClient, &http.Client{Transport: defaultTransport})
	}
	c := cfg.Client(ctx, token)
	c.Timeout = defaultHTTPTimeout
	return c
}
