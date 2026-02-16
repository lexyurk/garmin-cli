package main

import (
	"os"
	"sort"

	"github.com/lexyurk/garmin-cli/internal/output"
)

func renderKV(format, title string, fields map[string]string) error {
	switch format {
	case "markdown":
		return output.MarkdownKV(title, fields)
	case "human":
		return output.HumanTo(os.Stdout, title, fields)
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
		return output.TableTo(os.Stdout, []string{"field", "value"}, rows)
	default:
		return output.MarkdownKV(title, fields)
	}
}

func renderTable(format string, headers []string, rows [][]string) error {
	switch format {
	case "markdown":
		return output.MarkdownTable(headers, rows)
	case "table", "human":
		return output.TableTo(os.Stdout, headers, rows)
	default:
		return output.MarkdownTable(headers, rows)
	}
}

