// Package auth handles Garmin SSO authentication, token management and refresh.
package auth

// TODO: Implement Garmin SSO OAuth flow
// TODO: Token storage (~/.config/garmin-cli/tokens/)
// TODO: Token refresh logic
// TODO: MFA support

// Session holds authentication state for a Garmin Connect session.
type Session struct {
	Email       string
	AccessToken string
	// TODO: add refresh token, expiry, etc.
}

// Login authenticates with Garmin Connect using email/password via SSO.
func Login(email, password string) (*Session, error) {
	// TODO: implement Garmin SSO authentication flow
	return nil, nil
}

// Refresh refreshes an existing session's tokens.
func Refresh(s *Session) error {
	// TODO: implement token refresh
	return nil
}

// LoadSession loads a stored session from disk.
func LoadSession(profile string) (*Session, error) {
	// TODO: load tokens from config dir
	return nil, nil
}

// SaveSession persists session tokens to disk.
func SaveSession(s *Session, profile string) error {
	// TODO: save tokens to config dir
	return nil
}

// Logout clears stored tokens for a profile.
func Logout(profile string) error {
	// TODO: remove stored tokens
	return nil
}
