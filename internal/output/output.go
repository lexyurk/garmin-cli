// Package output handles formatting CLI output in various formats.
// Supports: Markdown (default), JSON, table, human-readable.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// JSONTo outputs data as formatted JSON to w.
func JSONTo(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// MarkdownTo writes Markdown text to w.
func MarkdownTo(w io.Writer, text string) error {
	_, err := io.WriteString(w, ensureTrailingNewline(text))
	return err
}

// MarkdownKVTo renders a stable key/value section to w.
func MarkdownKVTo(w io.Writer, title string, fields map[string]string) error {
	var b strings.Builder
	if title != "" {
		b.WriteString("## ")
		b.WriteString(title)
		b.WriteString("\n\n")
	}

	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := fields[k]
		b.WriteString("- **")
		b.WriteString(escapeMarkdownInline(k))
		b.WriteString("**: ")
		b.WriteString(escapeMarkdownInline(v))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	return MarkdownTo(w, b.String())
}

// MarkdownTableTo renders a GitHub-flavored Markdown table to w.
func MarkdownTableTo(w io.Writer, headers []string, rows [][]string) error {
	var b strings.Builder

	writeRow := func(colCount int, cols []string) {
		b.WriteString("|")
		for i := 0; i < colCount; i++ {
			col := ""
			if i < len(cols) {
				col = cols[i]
			}
			b.WriteString(" ")
			b.WriteString(escapeMarkdownInline(col))
			b.WriteString(" |")
		}
		b.WriteString("\n")
	}

	colCount := len(headers)
	writeRow(colCount, headers)
	sep := make([]string, len(headers))
	for i := range sep {
		sep[i] = "---"
	}
	writeRow(colCount, sep)
	for _, row := range rows {
		writeRow(colCount, row)
	}
	b.WriteString("\n")

	return MarkdownTo(w, b.String())
}

// TableTo outputs data as an aligned table to w.
func TableTo(w io.Writer, headers []string, rows [][]string) error {
	if len(headers) == 0 {
		return nil
	}

	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i := range widths {
			if i >= len(row) {
				continue
			}
			if l := len(row[i]); l > widths[i] {
				widths[i] = l
			}
		}
	}

	writeRow := func(cols []string) error {
		for i := range widths {
			val := ""
			if i < len(cols) {
				val = cols[i]
			}
			if i == len(widths)-1 {
				_, err := fmt.Fprintf(w, "%-*s", widths[i], val)
				return err
			}
			if _, err := fmt.Fprintf(w, "%-*s  ", widths[i], val); err != nil {
				return err
			}
		}
		return nil
	}

	if err := writeRow(headers); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}

	// separator
	sep := make([]string, len(widths))
	for i, w := range widths {
		sep[i] = strings.Repeat("-", w)
	}
	if err := writeRow(sep); err != nil {
		return err
	}
	if _, err := io.WriteString(w, "\n"); err != nil {
		return err
	}

	for _, row := range rows {
		if err := writeRow(row); err != nil {
			return err
		}
		if _, err := io.WriteString(w, "\n"); err != nil {
			return err
		}
	}
	return nil
}

// HumanTo outputs data in a simple human-readable format to w.
func HumanTo(w io.Writer, title string, fields map[string]string) error {
	if title != "" {
		if _, err := fmt.Fprintf(w, "=== %s ===\n", title); err != nil {
			return err
		}
	}
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if _, err := fmt.Fprintf(w, "  %s: %s\n", k, fields[k]); err != nil {
			return err
		}
	}
	return nil
}

func ensureTrailingNewline(s string) string {
	if s == "" {
		return "\n"
	}
	if strings.HasSuffix(s, "\n") {
		return s
	}
	return s + "\n"
}

func escapeMarkdownInline(s string) string {
	// Keep it simple: escape pipe and newlines for table cells.
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", "\\|")
	return s
}
