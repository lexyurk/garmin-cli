package main

import (
	"io"
	"os"
	"sort"

	"github.com/lexyurk/garmin-cli/internal/output"
)

func renderKV(format, title string, fields map[string]string) error {
	return renderKVTo(os.Stdout, format, title, fields)
}

func renderKVTo(w io.Writer, format, title string, fields map[string]string) error {
	switch format {
	case "markdown":
		return output.MarkdownKVTo(w, title, fields)
	case "human":
		return output.HumanTo(w, title, fields)
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
		return output.TableTo(w, []string{"field", "value"}, rows)
	default:
		return output.MarkdownKVTo(w, title, fields)
	}
}

func renderTable(format string, headers []string, rows [][]string) error {
	return renderTableTo(os.Stdout, format, headers, rows)
}

func renderTableTo(w io.Writer, format string, headers []string, rows [][]string) error {
	switch format {
	case "markdown":
		return output.MarkdownTableTo(w, headers, rows)
	case "table", "human":
		return output.TableTo(w, headers, rows)
	default:
		return output.MarkdownTableTo(w, headers, rows)
	}
}
