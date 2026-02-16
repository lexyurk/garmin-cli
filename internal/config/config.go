package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	AppName = "garmin"
)

var profileSanitizer = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

var ErrInvalidProfileName = errors.New("invalid profile name")

func ResolveConfigDir(flagValue string) (string, error) {
	if flagValue != "" {
		return cleanPath(flagValue)
	}
	if env := os.Getenv("GARMIN_CONFIG_DIR"); env != "" {
		return cleanPath(env)
	}

	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, AppName), nil
}

func TokensDir(configDir, profile string) string {
	return filepath.Join(configDir, "tokens", sanitizeProfile(profile))
}

func CacheDir(configDir string) string {
	return filepath.Join(configDir, "cache")
}

func OAuthConsumerCachePath(configDir string) string {
	return filepath.Join(CacheDir(configDir), "oauth_consumer.json")
}

func OAuth1TokenPath(configDir, profile string) string {
	return filepath.Join(TokensDir(configDir, profile), "oauth1_token.json")
}

func OAuth2TokenPath(configDir, profile string) string {
	return filepath.Join(TokensDir(configDir, profile), "oauth2_token.json")
}

func EnsurePrivateDir(path string) error {
	// Best effort on permissions; Windows will ignore modes.
	if err := os.MkdirAll(path, 0o700); err != nil {
		return err
	}
	return nil
}

func WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	if err := EnsurePrivateDir(dir); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, path)
}

func ReadFile(path string, maxBytes int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if maxBytes <= 0 {
		return io.ReadAll(f)
	}
	return io.ReadAll(io.LimitReader(f, maxBytes))
}

func cleanPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", errors.New("empty path")
	}
	if strings.HasPrefix(p, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		if p == "~" {
			p = home
		} else if strings.HasPrefix(p, "~/") {
			p = filepath.Join(home, p[2:])
		}
	}
	p = filepath.Clean(p)
	return p, nil
}

// ValidateProfile checks whether a user-provided profile name is safe to use as a directory name.
//
// Note: profile names are *sanitized* (see sanitizeProfile) before being used in paths. This
// validation only rejects names that would resolve to "." or ".." (which could otherwise make
// filepath.Join escape the intended directory).
func ValidateProfile(profile string) error {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return nil
	}

	base := filepath.Base(profile)
	// filepath.Base returns "." for "" and "."; for non-empty input we only care about "." / "..".
	if base == "." || base == ".." {
		return fmt.Errorf("%w: %q (resolves to %q)", ErrInvalidProfileName, profile, base)
	}

	// Defense-in-depth: reject if sanitization yields dot segments (should be impossible unless base is).
	sanitized := profileSanitizer.ReplaceAllString(base, "-")
	if sanitized == "." || sanitized == ".." || sanitized == "" {
		return fmt.Errorf("%w: %q (resolves to %q)", ErrInvalidProfileName, profile, sanitized)
	}
	return nil
}

func sanitizeProfile(profile string) string {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return "default"
	}
	profile = filepath.Base(profile) // prevent path traversal / separators
	if profile == "." || profile == ".." {
		return "default"
	}
	profile = profileSanitizer.ReplaceAllString(profile, "-")
	if profile == "" || profile == "." || profile == ".." {
		return "default"
	}
	return profile
}
