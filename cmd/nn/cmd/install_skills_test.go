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
		"nn-workflow", "nn-guide", "nn-navigate",
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

	stubPath := filepath.Join(destDir, "nn", "SKILL.md")
	data, err := os.ReadFile(stubPath)
	if err != nil {
		t.Fatalf("stub not created at %s: %v", stubPath, err)
	}
	if !strings.Contains(string(data), "nn skills get") {
		t.Errorf("stub at %s does not instruct Claude to call 'nn skills get': %q", stubPath, string(data))
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

func TestInstallSkillsForPi(t *testing.T) {
	_, execute := setupNotebook(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	_, err := execute("install-skills", "--for", "pi", "--list")
	if err != nil {
		t.Fatalf("nn install-skills --for pi --list: %v", err)
	}
}

// TestInstallSkillsMatchInstalled asserts that the installed nn stub at
// ~/.claude/skills/nn/SKILL.md instructs Claude to call 'nn skills get'.
// Skips if the stub is not installed.
func TestInstallSkillsMatchInstalled(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no HOME directory available")
	}
	stubPath := filepath.Join(home, ".claude", "skills", "nn", "SKILL.md")
	data, err := os.ReadFile(stubPath)
	if os.IsNotExist(err) {
		t.Skip("nn stub not installed at " + stubPath)
	}
	if err != nil {
		t.Fatalf("read installed stub: %v", err)
	}
	if !strings.Contains(string(data), "nn skills get") {
		t.Errorf("installed stub at %s does not instruct Claude to call 'nn skills get': %q", stubPath, string(data))
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
	// Skills step should show the stub path; hooks step should show hook output.
	if !strings.Contains(out, "nn/SKILL.md") {
		t.Errorf("nn install: expected stub path in output, got: %q", out)
	}
}

func TestInstallMetaCmdCopiesSkills(t *testing.T) {
	_, execute := setupNotebook(t)
	destDir := t.TempDir()

	_, err := execute("install", "--dest", destDir)
	if err != nil {
		t.Fatalf("nn install --dest: %v", err)
	}
	stubPath := filepath.Join(destDir, "nn", "SKILL.md")
	data, err := os.ReadFile(stubPath)
	if err != nil {
		t.Fatalf("stub not created at %s: %v", stubPath, err)
	}
	if !strings.Contains(string(data), "nn skills get") {
		t.Errorf("stub at %s does not instruct Claude to call 'nn skills get': %q", stubPath, string(data))
	}
}

func TestInstallForPiCopiesExtensionsAndSkills(t *testing.T) {
	_, execute := setupNotebook(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	out, err := execute("install", "--for", "pi")
	if err != nil {
		t.Fatalf("nn install --for pi: %v", err)
	}
	if !strings.Contains(out, "nn Pi extensions installed") {
		t.Fatalf("install --for pi output missing extensions message: %q", out)
	}

	extensionPath := filepath.Join(home, ".pi", "agent", "extensions", "nn_global_context.ts")
	if _, err := os.Stat(extensionPath); err != nil {
		t.Fatalf("Pi extension not installed at %s: %v", extensionPath, err)
	}

	stubPath := filepath.Join(home, ".pi", "agent", "skills", "nn", "SKILL.md")
	data, err := os.ReadFile(stubPath)
	if err != nil {
		t.Fatalf("Pi stub not installed at %s: %v", stubPath, err)
	}
	if !strings.Contains(string(data), "nn skills get") {
		t.Errorf("Pi stub does not instruct Claude to call 'nn skills get': %q", string(data))
	}
}
