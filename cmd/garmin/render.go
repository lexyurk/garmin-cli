package main

import (
	"io"
	"os"

	"github.com/lexyurk/garmin-cli/internal/output"
)

func renderKV(format, title string, fields map[string]string) error {
	return renderKVTo(os.Stdout, format, title, fields)
}

func renderKVTo(w io.Writer, format, title string, fields map[string]string) error {
	return output.RenderKVTo(w, format, title, fields)
}

func renderTable(format string, headers []string, rows [][]string) error {
	return renderTableTo(os.Stdout, format, headers, rows)
}

func renderTableTo(w io.Writer, format string, headers []string, rows [][]string) error {
	return output.RenderTableTo(w, format, headers, rows)
}
