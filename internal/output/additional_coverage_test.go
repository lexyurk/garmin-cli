package output

import (
	"errors"
	"strings"
	"testing"
)

type failAtWriter struct {
	calls  int
	failAt int
}

func (w *failAtWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == w.failAt {
		return 0, errors.New("write failed")
	}
	return len(p), nil
}

func TestOutputFunctions_PropagateWriterErrors(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(*failAtWriter) error
	}{
		{"json", func(w *failAtWriter) error { return JSONTo(w, map[string]int{"a": 1}) }},
		{"markdown", func(w *failAtWriter) error { return MarkdownTo(w, "text") }},
		{"markdown kv", func(w *failAtWriter) error { return MarkdownKVTo(w, "title", map[string]string{"a": "b"}) }},
		{"markdown table", func(w *failAtWriter) error { return MarkdownTableTo(w, []string{"a"}, [][]string{{"b"}}) }},
		{"human title", func(w *failAtWriter) error { return HumanTo(w, "title", nil) }},
		{"human field", func(w *failAtWriter) error { return HumanTo(w, "", map[string]string{"a": "b"}) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(&failAtWriter{failAt: 1}); err == nil {
				t.Fatal("expected writer error")
			}
		})
	}
}

func TestTableTo_LayoutEdgesAndEveryWriteFailure(t *testing.T) {
	if err := TableTo(&strings.Builder{}, nil, nil); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if err := TableTo(&out, []string{"a", "b"}, [][]string{{"longer", "x"}, {"short"}}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "longer") {
		t.Fatalf("unexpected table: %s", out.String())
	}

	// Two columns produce nine writes: two header cells + newline, two separator
	// cells + newline, then two data cells + newline.
	for failAt := 1; failAt <= 9; failAt++ {
		w := &failAtWriter{failAt: failAt}
		if err := TableTo(w, []string{"a", "b"}, [][]string{{"c", "d"}}); err == nil {
			t.Fatalf("expected failure at write %d", failAt)
		}
	}
}

func TestRenderAliasesDefaultsAndNewlineEdges(t *testing.T) {
	for _, format := range []string{"human", "md", "unknown"} {
		var out strings.Builder
		if err := RenderKVTo(&out, format, "title", map[string]string{"a": "b"}); err != nil || out.Len() == 0 {
			t.Fatalf("format %q output=%q err=%v", format, out.String(), err)
		}
	}
	for _, format := range []string{"tbl", "human", "md", "unknown"} {
		var out strings.Builder
		if err := RenderTableTo(&out, format, []string{"a"}, [][]string{{"b"}}); err != nil || out.Len() == 0 {
			t.Fatalf("format %q output=%q err=%v", format, out.String(), err)
		}
	}
	if ensureTrailingNewline("") != "\n" || ensureTrailingNewline("x\n") != "x\n" || ensureTrailingNewline("x") != "x\n" {
		t.Fatal("unexpected newline normalization")
	}
}
