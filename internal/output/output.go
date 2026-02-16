// Package output handles formatting CLI output in various formats.
// Supports: JSON, table, human-readable.
package output

import (
	"encoding/json"
	"fmt"
	"os"
)

// Format represents the output format type.
type Format string

const (
	FormatJSON  Format = "json"
	FormatTable Format = "table"
	FormatHuman Format = "human"
)

// JSON outputs data as formatted JSON to stdout.
func JSON(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
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
