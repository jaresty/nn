package main

import (
	"os"

	"github.com/jaresty/nn/cmd/nn/cmd"
)

// Injected at build time by GoReleaser ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	root := cmd.NewRootCmd("")
	root.Version = version + " (" + commit[:min(7, len(commit))] + " " + date + ")"
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
