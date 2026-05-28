package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestJSONTo_IndentedAndTrailingNewline(t *testing.T) {
	type payload struct {
		A int    `json:"a"`
		B string `json:"b"`
	}

	var b bytes.Buffer
	if err := JSONTo(&b, payload{A: 1, B: "x"}); err != nil {
		t.Fatalf("JSONTo error: %v", err)
	}
	got := b.String()
	if !strings.HasSuffix(got, "\n") {
		t.Fatalf("expected trailing newline, got: %q", got)
	}
	if !strings.Contains(got, "\n  \"a\": 1") || !strings.Contains(got, "\n  \"b\": \"x\"") {
		t.Fatalf("expected indented json, got:\n%s", got)
	}
}

func TestRenderTableTo_MarkdownAndTable(t *testing.T) {
	headers := []string{"h1", "h2"}
	rows := [][]string{{"a", "b"}}

	var md bytes.Buffer
	if err := RenderTableTo(&md, "markdown", headers, rows); err != nil {
		t.Fatalf("RenderTableTo markdown error: %v", err)
	}
	if !strings.Contains(md.String(), "| h1 | h2 |") {
		t.Fatalf("expected markdown table output, got:\n%s", md.String())
	}

	var tbl bytes.Buffer
	if err := RenderTableTo(&tbl, "table", headers, rows); err != nil {
		t.Fatalf("RenderTableTo table error: %v", err)
	}
	if !strings.Contains(tbl.String(), "h1") || !strings.Contains(tbl.String(), "h2") || !strings.Contains(tbl.String(), "-") {
		t.Fatalf("expected aligned table output, got:\n%s", tbl.String())
	}
}
