package main

import (
	"os"
)

var version = "dev"

func main() {
	if err := NewRootCmd(version).Execute(); err != nil {
		os.Exit(1)
	}
}
