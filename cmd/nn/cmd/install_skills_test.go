package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallSkillsList(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("install-skills", "--list")
	if err != nil {
		t.Fatalf("nn install-skills --list: %v", err)
	}
	if !strings.Contains(out, "nn-workflow") || !strings.Contains(out, "nn-guide") {
		t.Errorf("install-skills --list missing skill names: %q", out)
	}
}

func TestInstallSkillsCopies(t *testing.T) {
	_, execute := setupNotebook(t)
	destDir := t.TempDir()

	_, err := execute("install-skills", "--dest", destDir)
	if err != nil {
		t.Fatalf("nn install-skills: %v", err)
	}

	for _, skill := range []string{"nn-workflow", "nn-guide"} {
		skillPath := filepath.Join(destDir, skill, "SKILL.md")
		if _, err := os.Stat(skillPath); err != nil {
			t.Errorf("skill %s not installed at %s: %v", skill, skillPath, err)
		}
	}
}
