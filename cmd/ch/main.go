// Package main is the entry point for the ch CLI.
package main

import (
	"fmt"
	"os"

	"github.com/dmora/ch/internal/cli"
)

// Version information set by ldflags at build time.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Set version info for CLI
	cli.Version = version
	if commit != "none" && len(commit) > 7 {
		cli.Version = fmt.Sprintf("%s (%s, %s)", version, commit[:7], date)
	}

	if err := cli.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
