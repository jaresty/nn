package cmd

import (
	"os"
	"reflect"
	"strings"
	"testing"

	nnSkills "github.com/jaresty/nn/skills"
)

func TestSkillsList(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("skills", "list")
	if err != nil {
		t.Fatalf("nn skills list: %v", err)
	}
	for _, skill := range []string{"nn-workflow", "nn-guide", "nn-navigate", "nn-capture-discipline", "nn-session-debrief"} {
		if !strings.Contains(out, skill) {
			t.Errorf("skills list: missing %q in output: %q", skill, out)
		}
	}
}

func TestSkillsGet(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("skills", "get", "nn-workflow")
	if err != nil {
		t.Fatalf("nn skills get nn-workflow: %v", err)
	}
	if !strings.Contains(out, "nn-workflow") {
		t.Errorf("skills get nn-workflow: expected skill content, got: %q", out)
	}
}

func TestSkillsGetUnknownErrors(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("skills", "get", "no-such-skill")
	if err == nil {
		t.Fatal("skills get no-such-skill: want error, got nil")
	}
}

func TestSkillsGetNoNameErrors(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("skills", "get")
	if err == nil {
		t.Fatal("skills get (no name): want error, got nil")
	}
}

func TestSkillsGetDefaultOutputRemainsExactCore(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("skills", "get", "nn-navigate")
	if err != nil {
		t.Fatalf("nn skills get nn-navigate: %v", err)
	}
	core, err := nnSkills.FS.ReadFile("nn-navigate/SKILL.md")
	if err != nil {
		t.Fatalf("read embedded core: %v", err)
	}
	if out != string(core) {
		t.Fatal("default skills get output must be the byte-exact core without inlined references")
	}
	if strings.Contains(out, "# Reference: Ask") {
		t.Fatal("default skills get unexpectedly inlined references")
	}
}

func TestSkillsGetListsReferencesWithStableSortedApplicability(t *testing.T) {
	_, execute := setupNotebook(t)
	first, err := execute("skills", "get", "nn-navigate", "--list-references")
	if err != nil {
		t.Fatalf("list references: %v", err)
	}
	second, err := execute("skills", "get", "nn-navigate", "--list-references")
	if err != nil {
		t.Fatalf("list references second run: %v", err)
	}
	if first != second {
		t.Fatalf("reference listing is unstable:\nfirst=%q\nsecond=%q", first, second)
	}

	var names []string
	for _, line := range strings.Split(strings.TrimSpace(first), "\n") {
		fields := strings.SplitN(line, "\t", 2)
		if len(fields) != 2 || strings.TrimSpace(fields[1]) == "" {
			t.Fatalf("reference line must contain name and applicability separated by a tab: %q", line)
		}
		names = append(names, fields[0])
	}
	want := []string{"ask", "integrate", "lenses", "movement", "presentation", "scan-and-routes", "state"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("reference names = %v, want stable sorted %v", names, want)
	}
}

func TestSkillsGetReferenceReturnsOnlyRequestedMarkdown(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("skills", "get", "nn-navigate", "--reference", "movement")
	if err != nil {
		t.Fatalf("get movement reference: %v", err)
	}
	want, err := nnSkills.FS.ReadFile("nn-navigate/references/movement.md")
	if err != nil {
		t.Fatalf("read embedded movement reference: %v", err)
	}
	if out != string(want) {
		t.Fatal("reference output must be the byte-exact requested Markdown file")
	}
	if strings.Contains(out, "# Reference: Ask") {
		t.Fatal("reference retrieval inlined an unrequested reference")
	}
}

func TestSkillsGetReferenceRejectsUnsafeUnknownAndNonMarkdownNames(t *testing.T) {
	_, execute := setupNotebook(t)
	for _, name := range []string{
		"", "..", "../presentation", "references/presentation", `references\presentation`,
		"presentation.md", "unknown", "notes.txt",
	} {
		t.Run(strings.NewReplacer("/", "_", `\`, "_").Replace(name), func(t *testing.T) {
			if _, err := execute("skills", "get", "nn-navigate", "--reference", name); err == nil {
				t.Fatalf("--reference %q: want rejection", name)
			}
		})
	}
}

func TestSkillsGetReferenceFlagsAreMutuallyExclusive(t *testing.T) {
	_, execute := setupNotebook(t)
	if _, err := execute("skills", "get", "nn-navigate", "--list-references", "--reference", "movement"); err == nil {
		t.Fatal("--list-references and --reference must be mutually exclusive")
	}
}

func TestSkillsGetListReferencesCompatibilityForSkillWithoutReferences(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("skills", "get", "nn-workflow", "--list-references")
	if err != nil {
		t.Fatalf("list references for legacy single-file skill: %v", err)
	}
	if out != "" {
		t.Fatalf("single-file skill reference listing = %q, want empty", out)
	}
}

func TestInstallSkillsOutputShowsStubPath(t *testing.T) {
	_, execute := setupNotebook(t)
	destDir := t.TempDir()

	out, err := execute("install-skills", "--dest", destDir)
	if err != nil {
		t.Fatalf("nn install-skills: %v", err)
	}
	if strings.Contains(out, "nn-workflow") {
		t.Errorf("install-skills output should not list per-skill names, got: %q", out)
	}
	if !strings.Contains(out, "nn/SKILL.md") {
		t.Errorf("install-skills output should show stub path containing 'nn/SKILL.md', got: %q", out)
	}
}

func TestInstallSkillsStub(t *testing.T) {
	_, execute := setupNotebook(t)
	destDir := t.TempDir()

	_, err := execute("install-skills", "--dest", destDir)
	if err != nil {
		t.Fatalf("nn install-skills: %v", err)
	}

	stubPath := destDir + "/nn/SKILL.md"
	data, readErr := os.ReadFile(stubPath)
	if readErr != nil {
		t.Fatalf("stub not created at %s: %v", stubPath, readErr)
	}
	if !strings.Contains(string(data), "nn skills get") {
		t.Errorf("stub at %s does not instruct Claude to call 'nn skills get': %q", stubPath, string(data))
	}
}

func TestInstallSkillsRemovesDeprecated(t *testing.T) {
	_, execute := setupNotebook(t)
	destDir := t.TempDir()

	// Pre-create a deprecated per-skill dir to simulate old install.
	oldSkillDir := destDir + "/nn-workflow"
	if err := os.MkdirAll(oldSkillDir+"/", 0o755); err != nil {
		t.Fatalf("setup deprecated dir: %v", err)
	}
	oldFile := oldSkillDir + "/SKILL.md"
	if err := os.WriteFile(oldFile, []byte("old"), 0o644); err != nil {
		t.Fatalf("setup deprecated file: %v", err)
	}

	_, err := execute("install-skills", "--dest", destDir)
	if err != nil {
		t.Fatalf("nn install-skills: %v", err)
	}

	if _, statErr := os.Stat(oldSkillDir); !os.IsNotExist(statErr) {
		t.Errorf("deprecated skill dir %s was not removed", oldSkillDir)
	}
}

func TestSkillDescriptionsLeadWithUseWhen(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("skills", "list")
	if err != nil {
		t.Fatalf("nn skills list: %v", err)
	}
	for _, skill := range []string{"nn-workflow", "nn-guide", "nn-navigate", "nn-capture-discipline", "nn-session-debrief", "nn-link-suggester", "nn-refine", "nn-refine-workflow"} {
		content, readErr := execute("skills", "get", skill)
		if readErr != nil {
			t.Fatalf("nn skills get %s: %v", skill, readErr)
		}
		// description field must start with "Use when"
		for _, line := range strings.Split(content, "\n") {
			if strings.HasPrefix(line, "description:") {
				if !strings.Contains(line, "Use when") {
					t.Errorf("skill %s: description does not lead with 'Use when': %q", skill, line)
				}
				break
			}
		}
	}
	// stub must gate on nn skills list, not list skills statically
	if !strings.Contains(out, "Use when") {
		t.Errorf("nn skills list output does not contain 'Use when' routing triggers: %q", out)
	}
	// descriptions must not tell Claude to use slash commands — those no longer exist as installed skills
	if strings.Contains(out, "Invoke with /nn-") {
		t.Errorf("nn skills list output contains stale slash-command invocation 'Invoke with /nn-': %q", out)
	}
}

func TestStubGatesOnSkillsList(t *testing.T) {
	_, execute := setupNotebook(t)
	destDir := t.TempDir()
	if _, err := execute("install-skills", "--dest", destDir); err != nil {
		t.Fatalf("nn install-skills: %v", err)
	}
	data, err := os.ReadFile(destDir + "/nn/SKILL.md")
	if err != nil {
		t.Fatalf("stub not found: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "nn skills list") {
		t.Errorf("stub does not instruct Claude to run 'nn skills list': %q", content)
	}
	if strings.Contains(content, "nn skills get nn-workflow") {
		t.Errorf("stub contains hardcoded 'nn skills get nn-workflow' — should use live output instead")
	}
}
