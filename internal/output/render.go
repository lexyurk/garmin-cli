package output

import (
	"io"
	"sort"
	"strings"
)

// RenderKVTo renders a small key/value object in a format-specific way.
// Note: JSON rendering is intentionally not handled here because callers often
// want stable, typed JSON structures rather than a string map.
func RenderKVTo(w io.Writer, format, title string, fields map[string]string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "human":
		return HumanTo(w, title, fields)
	case "table":
		keys := make([]string, 0, len(fields))
		for k := range fields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		rows := make([][]string, 0, len(keys))
		for _, k := range keys {
			rows = append(rows, []string{k, fields[k]})
		}
		return TableTo(w, []string{"field", "value"}, rows)
	case "markdown", "md", "":
		return MarkdownKVTo(w, title, fields)
	default:
		// Be liberal: unknown formats fall back to Markdown (LLM-friendly default).
		return MarkdownKVTo(w, title, fields)
	}
}

// RenderTableTo renders a table in a format-specific way.
// JSON rendering is intentionally not handled here (call JSONTo directly).
func RenderTableTo(w io.Writer, format string, headers []string, rows [][]string) error {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "table", "human":
		return TableTo(w, headers, rows)
	case "markdown", "md", "":
		return MarkdownTableTo(w, headers, rows)
	default:
		return MarkdownTableTo(w, headers, rows)
	}
}
