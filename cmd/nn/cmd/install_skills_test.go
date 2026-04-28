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
	for _, skill := range []string{
		"nn-workflow", "nn-guide",
		"nn-capture-discipline", "nn-link-suggester", "nn-refine",
		"nn-session-debrief", "nn-refine-workflow",
	} {
		if !strings.Contains(out, skill) {
			t.Errorf("install-skills --list missing skill %q in output: %q", skill, out)
		}
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

func TestInstallMetaCmd(t *testing.T) {
	_, execute := setupNotebook(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	out, err := execute("install", "--for", "claude")
	if err != nil {
		t.Fatalf("nn install: %v", err)
	}
	// Both skills and hooks steps should produce output.
	if !strings.Contains(out, "nn-workflow") {
		t.Errorf("nn install: expected skill names in output, got: %q", out)
	}
}

func TestInstallMetaCmdCopiesSkills(t *testing.T) {
	_, execute := setupNotebook(t)
	destDir := t.TempDir()

	_, err := execute("install", "--dest", destDir)
	if err != nil {
		t.Fatalf("nn install --dest: %v", err)
	}
	for _, skill := range []string{
		"nn-workflow", "nn-guide",
		"nn-capture-discipline", "nn-link-suggester", "nn-refine",
		"nn-session-debrief", "nn-refine-workflow",
	} {
		skillPath := filepath.Join(destDir, skill, "SKILL.md")
		if _, err := os.Stat(skillPath); err != nil {
			t.Errorf("skill %s not installed at %s: %v", skill, skillPath, err)
		}
	}
}
