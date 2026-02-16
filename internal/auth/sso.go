package auth

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
)

var (
	ErrMFARequired = errors.New("mfa required")

	csrfRe   = regexp.MustCompile(`name="_csrf"\s+value="(.+?)"`)
	titleRe  = regexp.MustCompile(`(?is)<title>(.+?)</title>`)
	ticketRe = regexp.MustCompile(`embed\?ticket=([^"]+)"`)
)

func login(ctx context.Context, configDir, email, password string, promptMFA func() (string, error)) (OAuth1Token, OAuth2Token, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return OAuth1Token{}, OAuth2Token{}, err
	}

	httpClient := &http.Client{
		Timeout: defaultHTTPTimeout,
		Jar:     jar,
	}

	ssoBase := "https://sso.garmin.com/sso"
	ssoEmbed := ssoBase + "/embed"

	embedParams := url.Values{
		"id":          {"gauth-widget"},
		"embedWidget": {"true"},
		"gauthHost":   {ssoBase},
	}
	signinParams := url.Values{
		"id":                          {"gauth-widget"},
		"embedWidget":                 {"true"},
		"gauthHost":                   {ssoEmbed},
		"service":                     {ssoEmbed},
		"source":                      {ssoEmbed},
		"redirectAfterAccountLoginUrl": {ssoEmbed},
		"redirectAfterAccountCreationUrl": {ssoEmbed},
	}

	var lastURL string

	// 1) Set cookies.
	if lastURL, _, err = doRequest(ctx, httpClient, http.MethodGet, ssoEmbed, embedParams, lastURL, nil); err != nil {
		return OAuth1Token{}, OAuth2Token{}, err
	}

	// 2) Get CSRF token.
	var signinHTML string
	if lastURL, signinHTML, err = doRequest(ctx, httpClient, http.MethodGet, ssoBase+"/signin", signinParams, lastURL, nil); err != nil {
		return OAuth1Token{}, OAuth2Token{}, err
	}
	csrf, err := extractCSRF(signinHTML)
	if err != nil {
		return OAuth1Token{}, OAuth2Token{}, err
	}

	// 3) Submit credentials.
	form := url.Values{
		"username": {email},
		"password": {password},
		"embed":    {"true"},
		"_csrf":    {csrf},
	}
	headers := http.Header{
		"Content-Type": {"application/x-www-form-urlencoded"},
	}
	if lastURL, signinHTML, err = doRequest(ctx, httpClient, http.MethodPost, ssoBase+"/signin", signinParams, lastURL, strings.NewReader(form.Encode()), headers); err != nil {
		return OAuth1Token{}, OAuth2Token{}, err
	}

	if strings.Contains(strings.ToLower(extractTitle(signinHTML)), "mfa") {
		if promptMFA == nil {
			return OAuth1Token{}, OAuth2Token{}, ErrMFARequired
		}
		code, err := promptMFA()
		if err != nil {
			return OAuth1Token{}, OAuth2Token{}, err
		}

		csrf, err := extractCSRF(signinHTML)
		if err != nil {
			return OAuth1Token{}, OAuth2Token{}, err
		}
		mfaForm := url.Values{
			"mfa-code": {strings.TrimSpace(code)},
			"embed":    {"true"},
			"_csrf":    {csrf},
			"fromPage": {"setupEnterMfaCode"},
		}
		if lastURL, signinHTML, err = doRequest(
			ctx,
			httpClient,
			http.MethodPost,
			ssoBase+"/verifyMFA/loginEnterMfaCode",
			signinParams,
			lastURL,
			strings.NewReader(mfaForm.Encode()),
			headers,
		); err != nil {
			return OAuth1Token{}, OAuth2Token{}, err
		}
	}

	ticket, err := extractTicket(signinHTML)
	if err != nil {
		return OAuth1Token{}, OAuth2Token{}, err
	}

	consumer, err := getOAuthConsumer(ctx, configDir, httpClient)
	if err != nil {
		return OAuth1Token{}, OAuth2Token{}, err
	}

	oauth1, err := getOAuth1Token(ctx, consumer, ticket)
	if err != nil {
		return OAuth1Token{}, OAuth2Token{}, err
	}
	oauth2, err := exchangeOAuth2(ctx, consumer, oauth1)
	if err != nil {
		return OAuth1Token{}, OAuth2Token{}, err
	}

	return oauth1, oauth2, nil
}

func doRequest(
	ctx context.Context,
	httpClient *http.Client,
	method string,
	baseURL string,
	query url.Values,
	referer string,
	body io.Reader,
	extraHeaders ...http.Header,
) (string, string, error) {
	u := baseURL
	if query != nil && len(query) > 0 {
		u = u + "?" + query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", "com.garmin.android.apps.connectmobile")
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	for _, h := range extraHeaders {
		for k, vals := range h {
			for _, v := range vals {
				req.Header.Add(k, v)
			}
		}
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 16<<10))
		return "", "", fmt.Errorf("%s %s failed: %s: %s", method, baseURL, resp.Status, strings.TrimSpace(string(b)))
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	return resp.Request.URL.String(), string(b), nil
}

func extractCSRF(html string) (string, error) {
	m := csrfRe.FindStringSubmatch(html)
	if len(m) < 2 {
		return "", errors.New("couldn't find CSRF token in SSO response")
	}
	return m[1], nil
}

func extractTitle(html string) string {
	m := titleRe.FindStringSubmatch(html)
	if len(m) < 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

func extractTicket(html string) (string, error) {
	m := ticketRe.FindStringSubmatch(html)
	if len(m) < 2 {
		return "", errors.New("couldn't find SSO ticket in response")
	}
	return strings.TrimSpace(m[1]), nil
}

