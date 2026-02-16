// Package output handles formatting CLI output in various formats.
// Supports: Markdown (default), JSON, table, human-readable.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// Format represents the output format type.
type Format string

const (
	FormatMarkdown Format = "markdown"
	FormatJSON     Format = "json"
	FormatTable    Format = "table"
	FormatHuman    Format = "human"
)

// JSON outputs data as formatted JSON to stdout.
func JSON(data any) error {
	return JSONTo(os.Stdout, data)
}

// JSONTo outputs data as formatted JSON to w.
func JSONTo(w io.Writer, data any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// Markdown writes Markdown text to stdout.
func Markdown(text string) error {
	return MarkdownTo(os.Stdout, text)
}

// MarkdownTo writes Markdown text to w.
func MarkdownTo(w io.Writer, text string) error {
	_, err := io.WriteString(w, ensureTrailingNewline(text))
	return err
}

// MarkdownKV renders a stable key/value section.
func MarkdownKV(title string, fields map[string]string) error {
	return MarkdownKVTo(os.Stdout, title, fields)
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

// MarkdownTable renders a GitHub-flavored Markdown table.
func MarkdownTable(headers []string, rows [][]string) error {
	return MarkdownTableTo(os.Stdout, headers, rows)
}

// MarkdownTableTo renders a GitHub-flavored Markdown table to w.
func MarkdownTableTo(w io.Writer, headers []string, rows [][]string) error {
	var b strings.Builder

	writeRow := func(cols []string) {
		b.WriteString("|")
		for _, col := range cols {
			b.WriteString(" ")
			b.WriteString(escapeMarkdownInline(col))
			b.WriteString(" |")
		}
		b.WriteString("\n")
	}

	writeRow(headers)
	sep := make([]string, len(headers))
	for i := range sep {
		sep[i] = "---"
	}
	writeRow(sep)
	for _, row := range rows {
		writeRow(row)
	}
	b.WriteString("\n")

	return MarkdownTo(w, b.String())
}

// Table outputs data as an aligned table.
func Table(headers []string, rows [][]string) {
	// TODO: implement aligned table output with configurable columns
	for _, h := range headers {
		fmt.Printf("%-20s", h)
	}
	fmt.Println()
	for _, row := range rows {
		for _, col := range row {
			fmt.Printf("%-20s", col)
		}
		fmt.Println()
	}
}

// Human outputs data in a human-readable format.
func Human(title string, fields map[string]string) {
	// TODO: implement colorized human-readable output
	fmt.Printf("=== %s ===\n", title)
	for k, v := range fields {
		fmt.Printf("  %s: %s\n", k, v)
	}
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
