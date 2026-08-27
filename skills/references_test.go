package skills

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestReferencesFilesystemSourceIsSortedAndSafe(t *testing.T) {
	root := t.TempDir()
	writeTestSkill(t, root, "alpha", map[string]string{
		"zeta.md":   referenceFixture("when zeta applies", "zeta body"),
		"alpha.md":  referenceFixture("when alpha applies", "alpha body"),
		"notes.txt": "not a Markdown reference",
	})
	if err := os.Mkdir(filepath.Join(root, "alpha", "references", "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "alpha", "references", "nested", "hidden.md"), []byte(referenceFixture("never", "hidden")), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := ListReferences(os.DirFS(root), "alpha")
	if err != nil {
		t.Fatalf("ListReferences filesystem source: %v", err)
	}
	want := []Reference{
		{Name: "alpha", AppliesWhen: "when alpha applies"},
		{Name: "zeta", AppliesWhen: "when zeta applies"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("references = %#v, want %#v", got, want)
	}

	body, err := ReadReference(os.DirFS(root), "alpha", "zeta")
	if err != nil {
		t.Fatalf("ReadReference filesystem source: %v", err)
	}
	if string(body) != referenceFixture("when zeta applies", "zeta body") {
		t.Fatalf("reference body changed: %q", body)
	}
}

func TestReadReferenceRejectsUnsafeAndUnknownNames(t *testing.T) {
	root := t.TempDir()
	writeTestSkill(t, root, "alpha", map[string]string{
		"good.md":   referenceFixture("when needed", "good"),
		"notes.txt": "not Markdown",
	})
	fsys := os.DirFS(root)

	for _, name := range []string{
		"..", ".", "../outside", "nested/good", `nested\good`,
		"good.md", "notes", "notes.txt", "unknown", "UPPER", "two_words",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ReadReference(fsys, "alpha", name); err == nil {
				t.Fatalf("ReadReference(%q): want rejection", name)
			}
		})
	}

	for _, skill := range []string{"..", "../alpha", `nested\alpha`, "alpha/references", "alpha.md"} {
		t.Run("skill_"+strings.NewReplacer("/", "_", `\`, "_").Replace(skill), func(t *testing.T) {
			if _, err := ListReferences(fsys, skill); err == nil {
				t.Fatalf("ListReferences(%q): want rejection", skill)
			}
		})
	}
}

func TestReadReferenceDoesNotEagerlyParseUnrequestedReferences(t *testing.T) {
	root := t.TempDir()
	writeTestSkill(t, root, "alpha", map[string]string{
		"good.md": referenceFixture("when needed", "selected body"),
		"bad.md":  "---\napplies_when: [invalid, metadata]\n---\n",
	})

	data, err := ReadReference(os.DirFS(root), "alpha", "good")
	if err != nil {
		t.Fatalf("ReadReference parsed an unrequested reference: %v", err)
	}
	if !strings.Contains(string(data), "selected body") {
		t.Fatalf("ReadReference returned wrong content: %q", data)
	}
	if _, err := ListReferences(os.DirFS(root), "alpha"); err == nil {
		t.Fatal("ListReferences must still reject invalid applicability metadata")
	}
}

func TestReferencesRejectInvalidApplicabilityFrontmatter(t *testing.T) {
	fixtures := map[string]string{
		"blank-quoted":     "---\napplies_when: \"   \"\n---\n",
		"comment-only":     "---\napplies_when: # missing value\n---\n",
		"escaped-newline":  "---\napplies_when: \"first line\\nsecond line\"\n---\n",
		"block-scalar":     "---\napplies_when: |\n  one physical line\n---\n",
		"sequence":         "---\napplies_when: [first, second]\n---\n",
		"duplicate-fields": "---\napplies_when: first\napplies_when: second\n---\n",
	}
	for name, content := range fixtures {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeTestSkill(t, root, "alpha", map[string]string{"bad.md": content})
			if _, err := ListReferences(os.DirFS(root), "alpha"); err == nil {
				t.Fatalf("ListReferences accepted invalid applies_when frontmatter:\n%s", content)
			}
		})
	}
}

func TestReferencesRejectSymlinkEscapes(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "secret.md")
	if err := os.WriteFile(outsideFile, []byte(referenceFixture("secret", "must not escape")), 0o644); err != nil {
		t.Fatal(err)
	}

	writeTestSkill(t, root, "alpha", map[string]string{"good.md": referenceFixture("good", "good")})
	if err := os.Symlink(outsideFile, filepath.Join(root, "alpha", "references", "escape.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := ReadReference(os.DirFS(root), "alpha", "escape"); err == nil {
		t.Fatal("symlinked reference escaped the skill root")
	}

	outsideRefs := filepath.Join(outside, "references")
	if err := os.Mkdir(outsideRefs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outsideRefs, "escape.md"), []byte(referenceFixture("secret", "must not escape")), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "beta"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "beta", "SKILL.md"), []byte("# beta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideRefs, filepath.Join(root, "beta", "references")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := ListReferences(os.DirFS(root), "beta"); err == nil {
		t.Fatal("symlinked references directory escaped the skill root")
	}

	if err := os.Symlink(filepath.Join(root, "alpha"), filepath.Join(root, "gamma")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := ListReferences(os.DirFS(root), "gamma"); err == nil {
		t.Fatal("symlinked skill directory was accepted")
	}
}

func TestNNNavigateReferencesEmbeddedAndCopiedInstallation(t *testing.T) {
	wantNames := []string{"ask", "integrate", "lenses", "movement", "presentation", "scan-and-routes", "state"}

	embedded, err := ListReferences(FS, "nn-navigate")
	if err != nil {
		t.Fatalf("embedded references: %v", err)
	}
	if got := referenceNames(embedded); !reflect.DeepEqual(got, wantNames) {
		t.Fatalf("embedded reference names = %v, want %v", got, wantNames)
	}
	for _, ref := range embedded {
		data, err := ReadReference(FS, "nn-navigate", ref.Name)
		if err != nil {
			t.Fatalf("embedded reference %s: %v", ref.Name, err)
		}
		if len(data) == 0 {
			t.Fatalf("embedded reference %s is empty", ref.Name)
		}
	}

	installedRoot := t.TempDir()
	if err := CopySkillTree(FS, "nn-navigate", filepath.Join(installedRoot, "nn-navigate")); err != nil {
		t.Fatalf("copy installed skill tree: %v", err)
	}
	installed, err := ListReferences(os.DirFS(installedRoot), "nn-navigate")
	if err != nil {
		t.Fatalf("installed references: %v", err)
	}
	if !reflect.DeepEqual(installed, embedded) {
		t.Fatalf("installed references = %#v, embedded = %#v", installed, embedded)
	}
	for _, ref := range installed {
		installedData, err := ReadReference(os.DirFS(installedRoot), "nn-navigate", ref.Name)
		if err != nil {
			t.Fatalf("installed reference %s: %v", ref.Name, err)
		}
		embeddedData, _ := ReadReference(FS, "nn-navigate", ref.Name)
		if !reflect.DeepEqual(installedData, embeddedData) {
			t.Fatalf("installed reference %s differs from embedded source", ref.Name)
		}
	}
}

func writeTestSkill(t *testing.T, root, name string, refs map[string]string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Join(dir, "references"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("# "+name+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, content := range refs {
		if err := os.WriteFile(filepath.Join(dir, "references", name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func referenceFixture(appliesWhen, body string) string {
	return "---\napplies_when: \"" + appliesWhen + "\"\n---\n\n# Reference\n\n" + body + "\n"
}

func referenceNames(refs []Reference) []string {
	names := make([]string, len(refs))
	for i, ref := range refs {
		names[i] = ref.Name
	}
	return names
}
