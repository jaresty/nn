package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Wording regression guard, not an automated judgment of diagram quality.
func TestTranscriptSemanticVisualContract(t *testing.T) {
	root := filepath.Join("..", "..", "..", "skills", "nn-transcript")
	for file, terms := range map[string][]string{
		"SKILL.md":               {"Semantic thread layouts", "Declare both axes", "distinct job", "not a mandatory coordinate system", "Spawn topology remains authoritative"},
		"references/navigate.md": {"semantic thread-layout contract", "not arbitrary quadrants", "not independently verified", "No fixed axes are prescribed"},
	} {
		b, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatal(err)
		}
		for _, term := range terms {
			if !strings.Contains(string(b), term) {
				t.Errorf("%s missing visual contract: %q", file, term)
			}
		}
		for _, stale := range []string{"never the numbers or positions", "You may NEVER touch *position*", "never node position"} {
			if strings.Contains(string(b), stale) {
				t.Errorf("%s retains blanket position prohibition: %q", file, stale)
			}
		}
	}
}
