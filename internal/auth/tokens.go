package auth

import "time"

type OAuth1Token struct {
	OAuthToken       string `json:"oauth_token"`
	OAuthTokenSecret string `json:"oauth_token_secret"`
	MFAToken         string `json:"mfa_token,omitempty"`
	Domain           string `json:"domain,omitempty"`
}

type OAuth2Token struct {
	Scope                 string `json:"scope"`
	JTI                   string `json:"jti"`
	TokenType             string `json:"token_type"`
	AccessToken           string `json:"access_token"`
	RefreshToken          string `json:"refresh_token"`
	ExpiresIn             int    `json:"expires_in"`
	ExpiresAt             int64  `json:"expires_at"`
	RefreshTokenExpiresIn int    `json:"refresh_token_expires_in"`
	RefreshTokenExpiresAt int64  `json:"refresh_token_expires_at"`
}

func (t OAuth2Token) Expired(now time.Time) bool {
	// Add a small skew to avoid using tokens that are about to expire.
	const skewSeconds int64 = 30
	return now.Unix() >= (t.ExpiresAt - skewSeconds)
}
