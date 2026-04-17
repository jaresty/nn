package main

import (
	"strings"
	"testing"
)

func TestBuildVersionDevFallback(t *testing.T) {
	// When ldflags are not set (version == ""), buildVersion should not
	// return the old "dev (non none unknown)" string.
	got := buildVersion()
	if strings.Contains(got, "none") {
		t.Errorf("buildVersion() = %q — contains 'none', ldflags stub leaked into output", got)
	}
	if strings.Contains(got, "unknown") {
		t.Errorf("buildVersion() = %q — contains 'unknown', ldflags stub leaked into output", got)
	}
}

func TestBuildVersionLdflagsUsedWhenSet(t *testing.T) {
	// Temporarily set ldflags vars to simulate a GoReleaser build.
	orig := version
	origC := commit
	origD := date
	version = "v1.2.3"
	commit = "abcdef1234567"
	date = "2026-01-01"
	defer func() { version = orig; commit = origC; date = origD }()

	got := buildVersion()
	if !strings.Contains(got, "v1.2.3") {
		t.Errorf("buildVersion() = %q — expected v1.2.3", got)
	}
	if !strings.Contains(got, "abcdef1") {
		t.Errorf("buildVersion() = %q — expected truncated commit abcdef1", got)
	}
}
