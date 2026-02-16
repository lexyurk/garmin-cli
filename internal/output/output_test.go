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

func TestMarkdownTableTo_PadsMissingColumns(t *testing.T) {
	var b bytes.Buffer
	err := MarkdownTableTo(&b, []string{"a", "b"}, [][]string{
		{"only-a"},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	got := b.String()
	// Expect exactly 2 columns: "| only-a |  |"
	if !strings.Contains(got, "| only-a |  |") {
		t.Fatalf("expected padded empty column, got:\n%s", got)
	}
}

func TestMarkdownTableTo_TruncatesExtraColumns(t *testing.T) {
	var b bytes.Buffer
	err := MarkdownTableTo(&b, []string{"a", "b"}, [][]string{
		{"x", "y", "z"},
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	got := b.String()
	if strings.Contains(got, "z") {
		t.Fatalf("expected extra column to be truncated, got:\n%s", got)
	}
}

func TestHumanTo_SortsKeys(t *testing.T) {
	var b bytes.Buffer
	if err := HumanTo(&b, "Title", map[string]string{"b": "2", "a": "1"}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	got := b.String()
	aIdx := strings.Index(got, "a: 1")
	bIdx := strings.Index(got, "b: 2")
	if aIdx == -1 || bIdx == -1 {
		t.Fatalf("expected fields in output, got:\n%s", got)
	}
	if aIdx > bIdx {
		t.Fatalf("expected sorted keys, got:\n%s", got)
	}
}

func TestTableTo_WritesSeparator(t *testing.T) {
	var b bytes.Buffer
	if err := TableTo(&b, []string{"h1", "h2"}, [][]string{{"a", "bb"}}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	got := b.String()
	if !strings.Contains(got, "h1") || !strings.Contains(got, "h2") {
		t.Fatalf("expected headers, got:\n%s", got)
	}
	if !strings.Contains(got, "--") {
		t.Fatalf("expected separator line, got:\n%s", got)
	}
}

func TestRenderKVTo_Table_SortsKeys(t *testing.T) {
	var b bytes.Buffer
	if err := RenderKVTo(&b, "table", "ignored", map[string]string{"b": "2", "a": "1"}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	got := b.String()
	aIdx := strings.Index(got, "a")
	bIdx := strings.Index(got, "b")
	if aIdx == -1 || bIdx == -1 {
		t.Fatalf("expected keys in output, got:\n%s", got)
	}
	if aIdx > bIdx {
		t.Fatalf("expected sorted keys, got:\n%s", got)
	}
}

func TestRenderKVTo_TblAlias(t *testing.T) {
	var b bytes.Buffer
	if err := RenderKVTo(&b, "tbl", "ignored", map[string]string{"a": "1"}); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(b.String(), "field") || !strings.Contains(b.String(), "value") {
		t.Fatalf("expected table output, got:\n%s", b.String())
	}
}
