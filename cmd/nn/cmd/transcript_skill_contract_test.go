package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedTranscriptSkillSearchContractMatchesCLI(t *testing.T) {
	const assertion = "ASSERT_EMBEDDED_TRANSCRIPT_SKILL_SEARCH_CONTRACT_MATCHES_CLI"
	cmd := newTranscriptSearchCmd()
	for _, name := range []string{"session", "agent", "before", "raw", "json", "limit"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("%s: search flag --%s is absent", assertion, name)
		}
	}
	root := filepath.Join("..", "..", "..")
	paths := []string{
		filepath.Join(root, "skills", "nn-transcript", "SKILL.md"),
		filepath.Join(root, "skills", "nn-transcript", "references", "patterns.md"),
	}
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: read %s: %v", assertion, path, err)
		}
		text := string(body)
		if !strings.Contains(text, "nn transcript search") {
			t.Fatalf("%s: %s does not name the serving search command", assertion, path)
		}
	}
	core, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(core), "`nn grep") && !strings.Contains(string(core), "Never use `nn grep`") {
		t.Fatalf("%s: skill retains an nn grep instruction", assertion)
	}
}
