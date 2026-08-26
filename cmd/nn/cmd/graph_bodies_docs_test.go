package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// retained property [34]: public, virtual/global, and navigation-owner sources
// all teach topology-first, complete snapshot-bound body paging and the epistemic gate.
func TestGraphBodiesDocsAndNavigationProtocols(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	read := func(path string) string {
		t.Helper()
		data, err := os.ReadFile(filepath.Join(root, path))
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		return string(data)
	}

	for path, required := range map[string][]string{
		"README.md": {
			"nn graph bodies", "--page", "--snapshot", "48,000", "every page",
		},
		"skills/nn-guide/SKILL.md": {
			"### nn graph bodies", "--focus <id>", "--direction outgoing|incoming|both",
			"--links TYPE,...", "--status STATUS,...", "--representation VALUE",
			"48,000 bytes", "empty body", "stale or mismatched", "metadata boundary",
		},
		"skills/nn-navigate/SKILL.md": {
			"topology first", "nn graph show --focus <id> --depth 1 --direction both --zones --presentation-hints --color always --format text",
			"nn graph bodies --focus <id> --depth 1 --direction both --page 1",
			"every body page", "snapshot", "MUST NOT make body-derived claims",
		},
		"skills/nn-navigate/references/presentation.md": {
			"topology first", "nn graph bodies", "every page", "same snapshot",
			"body-derived central claim", "blocks presentation",
		},
	} {
		content := read(path)
		for _, snippet := range required {
			if !strings.Contains(content, snippet) {
				t.Errorf("%s missing graph body protocol %q", path, snippet)
			}
		}
	}

	legacyCanonical := "nn graph show --focus <id> --depth 1 --direction both --zones --bodies --presentation-hints --color always --format text"
	for _, path := range []string{"skills/nn-navigate/SKILL.md", "skills/nn-navigate/references/presentation.md"} {
		if strings.Contains(read(path), legacyCanonical) {
			t.Errorf("%s still uses deprecated inline bodies as canonical navigation read", path)
		}
	}

	_, execute := setupNotebook(t)
	virtual, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("show virtual CLI reference: %v", err)
	}
	for _, snippet := range []string{
		"**nn graph bodies**", "--page N", "--snapshot SHA256", "same filtered traversal set",
		"retrieve every page", "body-derived claims", "metadata boundary",
	} {
		if !strings.Contains(virtual, snippet) {
			t.Errorf("virtual/global CLI protocol missing %q", snippet)
		}
	}

	help, err := execute("graph", "bodies", "--help")
	if err != nil {
		t.Fatalf("graph bodies help: %v", err)
	}
	for _, snippet := range []string{"lossless", "JSON", "--page", "--snapshot", "--focus", "--depth", "--direction", "--links", "--status", "--representation"} {
		if !strings.Contains(help, snippet) {
			t.Errorf("graph bodies help missing %q:\n%s", snippet, help)
		}
	}
}
