package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestMarkdownKVTo_SortsKeys(t *testing.T) {
	var b bytes.Buffer
	err := MarkdownKVTo(&b, "X", map[string]string{
		"b": "2",
		"a": "1",
		"c": "3",
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	got := b.String()
	aIdx := strings.Index(got, "**a**")
	bIdx := strings.Index(got, "**b**")
	cIdx := strings.Index(got, "**c**")
	if aIdx == -1 || bIdx == -1 || cIdx == -1 {
		t.Fatalf("expected keys in output, got:\n%s", got)
	}
	if !(aIdx < bIdx && bIdx < cIdx) {
		t.Fatalf("expected sorted keys, got:\n%s", got)
	}
}

func TestMarkdownTableTo_EscapesPipesAndNewlines(t *testing.T) {
	var b bytes.Buffer
	err := MarkdownTableTo(&b, []string{"a|b", "c"}, [][]string{
		{"x|y", "line1\nline2"},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	got := b.String()
	if strings.Contains(got, "| a|b |") {
		t.Fatalf("expected header pipe to be escaped, got:\n%s", got)
	}
	if !strings.Contains(got, "a\\|b") {
		t.Fatalf("expected escaped header, got:\n%s", got)
	}
	if !strings.Contains(got, "x\\|y") {
		t.Fatalf("expected escaped cell, got:\n%s", got)
	}
	if strings.Contains(got, "line1\nline2") {
		t.Fatalf("expected newlines replaced in cell, got:\n%s", got)
	}
	if !strings.Contains(got, "line1 line2") {
		t.Fatalf("expected newline replaced with space, got:\n%s", got)
	}
}

