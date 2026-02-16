package config

import "testing"

func TestSanitizeProfile(t *testing.T) {
	cases := map[string]string{
		"":             "default",
		"   ":          "default",
		"default":      "default",
		"my-profile":   "my-profile",
		"../secret":    "secret",
		"../../x":      "x",
		"spaces here":  "spaces-here",
		"weird/thing":  "thing",
		"üñïçødê":      "-d-",
	}

	for in, want := range cases {
		got := sanitizeProfile(in)
		if got != want {
			t.Fatalf("sanitizeProfile(%q) = %q, want %q", in, got, want)
		}
	}
}

