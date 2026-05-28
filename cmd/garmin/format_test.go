package main

import "testing"

func TestNormalizeFormat_UnknownPassthrough(t *testing.T) {
	if got := normalizeFormat("WeIrD"); got != "weird" {
		t.Fatalf("unexpected normalizeFormat: %q", got)
	}
}
