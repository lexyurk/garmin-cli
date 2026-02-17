package main

import (
	"strings"
	"testing"
)

func TestReadLine_TrimsNewlineAndSpaces(t *testing.T) {
	got, err := readLine("", strings.NewReader("  hello world  \n"))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "hello world" {
		t.Fatalf("expected trimmed line, got %q", got)
	}
}

func TestReadLine_AllowsEOFWithoutNewline(t *testing.T) {
	got, err := readLine("", strings.NewReader("hello"))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != "hello" {
		t.Fatalf("expected line, got %q", got)
	}
}
