package main

import (
	"fmt"
	"os"
)

var version = "dev"

func main() {
	if err := NewRootCmd(version).Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
