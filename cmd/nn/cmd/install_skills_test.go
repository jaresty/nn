package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	nnSkills "github.com/jaresty/nn/skills"
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

// TestInstallSkillsMatchInstalled asserts that each embedded skill's SKILL.md
// matches the installed version at ~/.claude/skills/<skill>/SKILL.md.
// Skips individual skills that are not yet installed; skips entirely if HOME is not set.
func TestInstallSkillsMatchInstalled(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no HOME directory available")
	}
	installedBase := filepath.Join(home, ".claude", "skills")

	entries, err := nnSkills.FS.ReadDir(".")
	if err != nil {
		t.Fatalf("read embedded skills: %v", err)
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		installedPath := filepath.Join(installedBase, name, "SKILL.md")
		installedData, err := os.ReadFile(installedPath)
		if os.IsNotExist(err) {
			t.Logf("skill %s: not installed, skipping", name)
			continue
		}
		if err != nil {
			t.Errorf("skill %s: read installed: %v", name, err)
			continue
		}
		embeddedData, err := nnSkills.FS.ReadFile(filepath.Join(name, "SKILL.md"))
		if err != nil {
			t.Errorf("skill %s: read embedded: %v", name, err)
			continue
		}
		if string(embeddedData) != string(installedData) {
			t.Errorf("skill %s: embedded SKILL.md does not match installed at %s\n--- embedded ---\n%s\n--- installed ---\n%s",
				name, installedPath, embeddedData, installedData)
		}
	}
}

func TestInstallSkillsUnknownForErrors(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("install-skills", "--for", "unknownllm")
	if err == nil {
		t.Fatal("--for unknownllm: want error, got nil")
	}
}


// Assertion: nn-session-debrief SKILL.md contains --partial mode with partial-debrief tag and skips for steps 6/7.
func TestNNSessionDebriefPartialMode(t *testing.T) {
	data, err := nnSkills.FS.ReadFile("nn-session-debrief/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-session-debrief/SKILL.md: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "--partial") {
		t.Errorf("nn-session-debrief SKILL.md: expected '--partial' flag documented; got no match")
	}
	if !strings.Contains(content, "partial-debrief") {
		t.Errorf("nn-session-debrief SKILL.md: expected 'partial-debrief' tag instruction; got no match")
	}
	if !strings.Contains(content, "skip") || !strings.Contains(content, "step 6") {
		t.Errorf("nn-session-debrief SKILL.md: expected partial mode to document skipping step 6; got no match")
	}
}

// Assertion: TestNNSessionDebriefRelayReminder — --partial section instructs LLM to emit relay handoff reminder
func TestNNSessionDebriefRelayReminder(t *testing.T) {
	data, err := nnSkills.FS.ReadFile("nn-session-debrief/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-session-debrief/SKILL.md: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "Relay block not updated") {
		t.Errorf("nn-session-debrief SKILL.md: expected relay handoff reminder 'Relay block not updated' in --partial section; got no match")
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
