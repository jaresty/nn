package main

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/jaresty/nn/cmd/nn/cmd"
)

// Injected at build time by GoReleaser ldflags.
var (
	version = ""
	commit  = ""
	date    = ""
)

func main() {
	root := cmd.NewRootCmd("")
	root.Version = buildVersion()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// buildVersion returns a version string, preferring GoReleaser ldflags and
// falling back to VCS info embedded by go install / go build.
func buildVersion() string {
	if version != "" {
		c := commit
		if len(c) > 7 {
			c = c[:7]
		}
		return fmt.Sprintf("%s (%s %s)", version, c, date)
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		v := info.Main.Version
		if v == "" || v == "(devel)" {
			v = "dev"
		}
		var vcsRev, vcsTime string
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				if len(s.Value) > 7 {
					vcsRev = s.Value[:7]
				} else {
					vcsRev = s.Value
				}
			case "vcs.time":
				vcsTime = s.Value
			}
		}
		if vcsRev != "" {
			return fmt.Sprintf("%s (%s %s)", v, vcsRev, vcsTime)
		}
		return v
	}

	return "dev"
}
