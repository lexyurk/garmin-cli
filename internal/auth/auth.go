// Package auth implements Garmin SSO login and token persistence.
package auth

import (
	"context"
	"errors"
	"os"

	"github.com/lexyurk/garmin-cli/internal/config"
)

// Session contains OAuth credentials needed to call connectapi.garmin.com.
type Session struct {
	OAuth1 OAuth1Token
	OAuth2 OAuth2Token
}

var ErrNotAuthenticated = errors.New("not authenticated (run `garmin auth login`)")

// Login performs the full SSO → OAuth1 → OAuth2 flow.
func Login(ctx context.Context, configDir, email, password string, promptMFA func() (string, error)) (*Session, error) {
	oauth1, oauth2, err := login(ctx, configDir, email, password, promptMFA)
	if err != nil {
		return nil, err
	}
	return &Session{OAuth1: oauth1, OAuth2: oauth2}, nil
}

// RefreshOAuth2 exchanges the OAuth1 token for a fresh OAuth2 token.
func RefreshOAuth2(ctx context.Context, configDir string, oauth1 OAuth1Token) (OAuth2Token, error) {
	consumer, err := getOAuthConsumer(ctx, configDir, nil)
	if err != nil {
		return OAuth2Token{}, err
	}
	return exchangeOAuth2(ctx, consumer, oauth1)
}

func LoadSession(configDir, profile string) (*Session, error) {
	oauth1Path := config.OAuth1TokenPath(configDir, profile)
	oauth2Path := config.OAuth2TokenPath(configDir, profile)

	oauth1, err := loadJSON[OAuth1Token](oauth1Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotAuthenticated
		}
		return nil, err
	}
	oauth2, err := loadJSON[OAuth2Token](oauth2Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNotAuthenticated
		}
		return nil, err
	}

	return &Session{OAuth1: oauth1, OAuth2: oauth2}, nil
}

func SaveSession(configDir, profile string, s *Session) error {
	oauth1Path := config.OAuth1TokenPath(configDir, profile)
	oauth2Path := config.OAuth2TokenPath(configDir, profile)
	if err := saveJSON(oauth1Path, s.OAuth1, 0o600); err != nil {
		return err
	}
	if err := saveJSON(oauth2Path, s.OAuth2, 0o600); err != nil {
		return err
	}
	return nil
}

func Logout(configDir, profile string) error {
	dir := config.TokensDir(configDir, profile)
	return os.RemoveAll(dir)
}
