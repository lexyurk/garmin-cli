package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/lexyurk/garmin-cli/internal/config"
)

const (
	oauthConsumerURL  = "https://thegarth.s3.amazonaws.com/oauth_consumer.json"
	consumerCacheTTL  = 7 * 24 * time.Hour
	defaultHTTPTimeout = 20 * time.Second
)

type oauthConsumer struct {
	ConsumerKey    string `json:"consumer_key"`
	ConsumerSecret string `json:"consumer_secret"`
}

type oauthConsumerCache struct {
	ConsumerKey    string `json:"consumer_key"`
	ConsumerSecret string `json:"consumer_secret"`
	FetchedAt      int64  `json:"fetched_at"`
}

func getOAuthConsumer(ctx context.Context, configDir string, httpClient *http.Client) (oauthConsumer, error) {
	cachePath := config.OAuthConsumerCachePath(configDir)

	if cached, ok := loadCachedConsumer(cachePath); ok {
		return cached, nil
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultHTTPTimeout}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, oauthConsumerURL, nil)
	if err != nil {
		return oauthConsumer{}, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return oauthConsumer{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return oauthConsumer{}, fmt.Errorf("oauth consumer fetch failed: %s", resp.Status)
	}

	var raw oauthConsumer
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return oauthConsumer{}, err
	}

	if raw.ConsumerKey == "" || raw.ConsumerSecret == "" {
		return oauthConsumer{}, errors.New("oauth consumer response missing fields")
	}

	cache := oauthConsumerCache{
		ConsumerKey:    raw.ConsumerKey,
		ConsumerSecret: raw.ConsumerSecret,
		FetchedAt:      time.Now().Unix(),
	}
	_ = saveJSON(cachePath, cache, 0o600)

	return raw, nil
}

func loadCachedConsumer(path string) (oauthConsumer, bool) {
	cache, err := loadJSON[oauthConsumerCache](path)
	if err != nil {
		return oauthConsumer{}, false
	}

	if cache.ConsumerKey == "" || cache.ConsumerSecret == "" || cache.FetchedAt == 0 {
		return oauthConsumer{}, false
	}
	age := time.Since(time.Unix(cache.FetchedAt, 0))
	if age < 0 || age > consumerCacheTTL {
		return oauthConsumer{}, false
	}

	return oauthConsumer{
		ConsumerKey:    cache.ConsumerKey,
		ConsumerSecret: cache.ConsumerSecret,
	}, true
}

