package auth

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path string, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func TestLogout_RejectsDotDotAndDoesNotDeleteConfigDir(t *testing.T) {
	cfgDir := t.TempDir()

	marker := filepath.Join(cfgDir, "marker.txt")
	writeFile(t, marker, "keep")

	other := filepath.Join(cfgDir, "tokens", "other", "keep.txt")
	writeFile(t, other, "keep")

	if err := Logout(cfgDir, ".."); err == nil {
		t.Fatalf("expected error for profile %q", "..")
	}

	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("marker should still exist; stat: %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("other profile tokens should still exist; stat: %v", err)
	}
}

func TestLogout_RejectsDotAndDoesNotDeleteOtherProfiles(t *testing.T) {
	cfgDir := t.TempDir()

	other := filepath.Join(cfgDir, "tokens", "other", "keep.txt")
	writeFile(t, other, "keep")

	if err := Logout(cfgDir, "."); err == nil {
		t.Fatalf("expected error for profile %q", ".")
	}

	if _, err := os.Stat(other); err != nil {
		t.Fatalf("other profile tokens should still exist; stat: %v", err)
	}
}

func TestLogout_RemovesOnlyThatProfileDir(t *testing.T) {
	cfgDir := t.TempDir()

	marker := filepath.Join(cfgDir, "marker.txt")
	writeFile(t, marker, "keep")

	target := filepath.Join(cfgDir, "tokens", "work", "oauth1_token.json")
	writeFile(t, target, "{}")

	other := filepath.Join(cfgDir, "tokens", "other", "keep.txt")
	writeFile(t, other, "keep")

	if err := Logout(cfgDir, "work"); err != nil {
		t.Fatalf("Logout(work) unexpected error: %v", err)
	}

	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Fatalf("expected target token file to be removed; stat err=%v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("other profile tokens should still exist; stat: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("marker should still exist; stat: %v", err)
	}
}
