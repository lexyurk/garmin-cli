package main

import (
	"io"

	"github.com/lexyurk/garmin-cli/internal/output"
)

func renderKVTo(w io.Writer, format, title string, fields map[string]string) error {
	return output.RenderKVTo(w, format, title, fields)
}

func renderTableTo(w io.Writer, format string, headers []string, rows [][]string) error {
	return output.RenderTableTo(w, format, headers, rows)
}
