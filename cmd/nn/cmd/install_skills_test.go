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

func TestInstallSkillsForClaude(t *testing.T) {
	_, execute := setupNotebook(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := execute("install-skills", "--for", "claude", "--list")
	if err != nil {
		t.Fatalf("nn install-skills --for claude --list: %v", err)
	}
}

func TestInstallSkillsForCursor(t *testing.T) {
	_, execute := setupNotebook(t)
	destDir := t.TempDir()

	out, err := execute("install-skills", "--for", "cursor", "--dest", destDir, "--list")
	if err != nil {
		t.Fatalf("nn install-skills --for cursor --list: %v", err)
	}
	if !strings.Contains(out, "nn-workflow") {
		t.Errorf("--for cursor --list missing skill names: %q", out)
	}
}

func TestInstallSkillsUnknownForErrors(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("install-skills", "--for", "unknownllm")
	if err == nil {
		t.Fatal("--for unknownllm: want error, got nil")
	}
}
