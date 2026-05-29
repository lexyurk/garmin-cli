package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveConfigDir_FlagWins(t *testing.T) {
	t.Setenv("GARMIN_CONFIG_DIR", t.TempDir())

	got, err := ResolveConfigDir("  /tmp/garmin-test  ")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "/tmp/garmin-test" {
		t.Fatalf("expected cleaned flag path, got %q", got)
	}
}

func TestResolveConfigDir_EnvUsed(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("GARMIN_CONFIG_DIR", dir)

	got, err := ResolveConfigDir("")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != dir {
		t.Fatalf("expected env dir %q, got %q", dir, got)
	}
}

func TestResolveConfigDir_DefaultUsesUserConfigDir(t *testing.T) {
	// Keep the lookup hermetic on Linux (XDG_CONFIG_HOME is honored there).
	// On macOS os.UserConfigDir() ignores XDG and returns
	// ~/Library/Application Support, so derive the expected base from the same
	// source ResolveConfigDir uses instead of assuming XDG semantics.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GARMIN_CONFIG_DIR", "")

	base, err := os.UserConfigDir()
	if err != nil {
		t.Skipf("os.UserConfigDir() unavailable: %v", err)
	}

	got, err := ResolveConfigDir("")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want := filepath.Join(base, AppName)
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestCleanPath_TildeExpansion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	got, err := cleanPath("~")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != home {
		t.Fatalf("expected home %q, got %q", home, got)
	}

	got2, err := cleanPath("~/x/y")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	want2 := filepath.Join(home, "x", "y")
	if got2 != want2 {
		t.Fatalf("expected %q, got %q", want2, got2)
	}
}

func TestCleanPath_EmptyRejected(t *testing.T) {
	if _, err := cleanPath("   "); err == nil {
		t.Fatalf("expected error")
	}
}

func TestWriteFileAtomic_WritesAndSetsPerm(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "file.txt")

	if err := WriteFileAtomic(path, []byte("hello"), 0o600); err != nil {
		t.Fatalf("WriteFileAtomic error: %v", err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(b) != "hello" {
		t.Fatalf("unexpected contents: %q", string(b))
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("unexpected perm: %v", info.Mode().Perm())
	}
}

func TestReadFile_LimitsBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("abcdef"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	b, err := ReadFile(path, 3)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if string(b) != "abc" {
		t.Fatalf("expected limited read, got %q", string(b))
	}

	b2, err := ReadFile(path, 0)
	if err != nil {
		t.Fatalf("ReadFile error: %v", err)
	}
	if strings.TrimSpace(string(b2)) != "abcdef" {
		t.Fatalf("expected full read, got %q", string(b2))
	}
}
