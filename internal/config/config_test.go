package config

import "testing"

func TestSanitizeProfile(t *testing.T) {
	cases := map[string]string{
		"":            "default",
		"   ":         "default",
		".":           "default",
		"..":          "default",
		"./.":         "default",
		"./..":        "default",
		"a/.":         "default",
		"a/..":        "default",
		"default":     "default",
		"my-profile":  "my-profile",
		"../secret":   "secret",
		"../../x":     "x",
		"spaces here": "spaces-here",
		"weird/thing": "thing",
		"üñïçødê":     "-d-",
	}

	for in, want := range cases {
		got := sanitizeProfile(in)
		if got != want {
			t.Fatalf("sanitizeProfile(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestValidateProfile(t *testing.T) {
	ok := []string{
		"",
		"   ",
		"default",
		"my-profile",
		"../secret",
		"../../x",
		"spaces here",
		"weird/thing",
		"üñïçødê",
		"...",
		".a",
		"a..b",
	}
	for _, in := range ok {
		if err := ValidateProfile(in); err != nil {
			t.Fatalf("ValidateProfile(%q) unexpected error: %v", in, err)
		}
	}

	bad := []string{
		".",
		"..",
		"./.",
		"./..",
		"a/.",
		"a/..",
		"../..",
	}
	for _, in := range bad {
		if err := ValidateProfile(in); err == nil {
			t.Fatalf("ValidateProfile(%q) expected error, got nil", in)
		}
	}
}
