package skills

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Reference describes a lazily retrievable Markdown file owned by a skill.
type Reference struct {
	Name        string
	AppliesWhen string
}

// ReadSkill returns the skill's core SKILL.md selected by logical name.
func ReadSkill(fsys fs.FS, skill string) ([]byte, error) {
	if !validLogicalName(skill) {
		return nil, fmt.Errorf("invalid skill name %q", skill)
	}
	if err := requireDirectDir(fsys, ".", skill); err != nil {
		return nil, fmt.Errorf("skill %q not found", skill)
	}
	entries, err := fs.ReadDir(fsys, skill)
	if err != nil {
		return nil, fmt.Errorf("read skill %q: %w", skill, err)
	}
	for _, entry := range entries {
		if entry.Name() != "SKILL.md" {
			continue
		}
		if err := requireRegularEntry(entry); err != nil {
			return nil, fmt.Errorf("read skill %q: SKILL.md: %w", skill, err)
		}
		data, err := fs.ReadFile(fsys, path.Join(skill, "SKILL.md"))
		if err != nil {
			return nil, fmt.Errorf("read skill %q: %w", skill, err)
		}
		return data, nil
	}
	return nil, fmt.Errorf("skill %q not found", skill)
}

// ListReferences returns the skill's direct references/*.md files in stable
// lexical order. Skills without a references directory return an empty list.
func ListReferences(fsys fs.FS, skill string) ([]Reference, error) {
	if !validLogicalName(skill) {
		return nil, fmt.Errorf("invalid skill name %q", skill)
	}
	if err := requireDirectDir(fsys, ".", skill); err != nil {
		return nil, fmt.Errorf("skill %q not found", skill)
	}

	referencesDir := path.Join(skill, "references")
	if err := requireDirectDir(fsys, skill, "references"); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return []Reference{}, nil
		}
		return nil, fmt.Errorf("list references for skill %q: %w", skill, err)
	}

	entries, err := fs.ReadDir(fsys, referencesDir)
	if err != nil {
		return nil, fmt.Errorf("list references for skill %q: %w", skill, err)
	}
	refs := make([]Reference, 0, len(entries))
	for _, entry := range entries {
		fileName := entry.Name()
		if path.Ext(fileName) != ".md" {
			continue
		}
		name := strings.TrimSuffix(fileName, ".md")
		if !validLogicalName(name) {
			continue
		}
		if err := requireRegularEntry(entry); err != nil {
			return nil, fmt.Errorf("reference %q in skill %q: %w", name, skill, err)
		}
		data, err := fs.ReadFile(fsys, path.Join(referencesDir, fileName))
		if err != nil {
			return nil, fmt.Errorf("read reference %q in skill %q: %w", name, skill, err)
		}
		appliesWhen, err := referenceApplicability(data)
		if err != nil {
			return nil, fmt.Errorf("reference %q in skill %q: %w", name, skill, err)
		}
		refs = append(refs, Reference{Name: name, AppliesWhen: appliesWhen})
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Name < refs[j].Name })
	return refs, nil
}

// ReadReference returns one direct Markdown reference selected by logical name.
// The caller cannot supply a path or file extension.
func ReadReference(fsys fs.FS, skill, name string) ([]byte, error) {
	if !validLogicalName(skill) {
		return nil, fmt.Errorf("invalid skill name %q", skill)
	}
	if !validLogicalName(name) {
		return nil, fmt.Errorf("invalid reference name %q", name)
	}
	if err := requireDirectDir(fsys, ".", skill); err != nil {
		return nil, fmt.Errorf("skill %q not found", skill)
	}
	if err := requireDirectDir(fsys, skill, "references"); err != nil {
		return nil, fmt.Errorf("reference %q not found for skill %q", name, skill)
	}

	referencesDir := path.Join(skill, "references")
	entries, err := fs.ReadDir(fsys, referencesDir)
	if err != nil {
		return nil, fmt.Errorf("read reference %q for skill %q: %w", name, skill, err)
	}
	fileName := name + ".md"
	for _, entry := range entries {
		if entry.Name() != fileName {
			continue
		}
		if err := requireRegularEntry(entry); err != nil {
			return nil, fmt.Errorf("invalid reference %q for skill %q", name, skill)
		}
		data, err := fs.ReadFile(fsys, path.Join(referencesDir, fileName))
		if err != nil {
			return nil, fmt.Errorf("read reference %q for skill %q: %w", name, skill, err)
		}
		if _, err := referenceApplicability(data); err != nil {
			return nil, fmt.Errorf("invalid reference %q for skill %q", name, skill)
		}
		return data, nil
	}
	return nil, fmt.Errorf("reference %q not found for skill %q", name, skill)
}

// CopySkillTree recursively materializes one embedded or filesystem-backed skill,
// preserving reference subdirectories. Symlinks and non-regular files are rejected.
func CopySkillTree(fsys fs.FS, skill, destDir string) error {
	if !validLogicalName(skill) {
		return fmt.Errorf("invalid skill name %q", skill)
	}
	if err := requireDirectDir(fsys, ".", skill); err != nil {
		return fmt.Errorf("skill %q not found", skill)
	}
	return fs.WalkDir(fsys, skill, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("copy skill %q: symlink %q is not allowed", skill, sourcePath)
		}
		rel := strings.TrimPrefix(sourcePath, skill)
		rel = strings.TrimPrefix(rel, "/")
		destination := destDir
		if rel != "" {
			destination = filepath.Join(destDir, filepath.FromSlash(rel))
		}
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o755)
		}
		if err := requireRegularEntry(entry); err != nil {
			return fmt.Errorf("copy skill %q: %s: %w", skill, sourcePath, err)
		}
		data, err := fs.ReadFile(fsys, sourcePath)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, data, 0o644)
	})
}

func validLogicalName(name string) bool {
	if name == "" || name[0] == '-' || name[len(name)-1] == '-' {
		return false
	}
	previousHyphen := false
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			previousHyphen = false
		case r == '-' && !previousHyphen:
			previousHyphen = true
		default:
			return false
		}
	}
	return true
}

func requireDirectDir(fsys fs.FS, parent, name string) error {
	entries, err := fs.ReadDir(fsys, parent)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() != name {
			continue
		}
		if entry.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("%s is a symlink", path.Join(parent, name))
		}
		if !entry.IsDir() {
			return fmt.Errorf("%s is not a directory", path.Join(parent, name))
		}
		return nil
	}
	return fs.ErrNotExist
}

func requireRegularEntry(entry fs.DirEntry) error {
	if entry.Type()&fs.ModeSymlink != 0 {
		return errors.New("symlinks are not allowed")
	}
	info, err := entry.Info()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("not a regular file")
	}
	return nil
}

func referenceApplicability(data []byte) (string, error) {
	lines := strings.Split(string(data), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", errors.New("missing frontmatter")
	}
	end := -1
	for i, line := range lines[1:] {
		if strings.TrimSpace(line) == "---" {
			end = i + 1
			break
		}
	}
	if end < 0 {
		return "", errors.New("missing closing frontmatter delimiter")
	}

	var document yaml.Node
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &document); err != nil {
		return "", fmt.Errorf("invalid frontmatter: %w", err)
	}
	if len(document.Content) != 1 || document.Content[0].Kind != yaml.MappingNode {
		return "", errors.New("frontmatter must be a mapping")
	}

	var appliesWhen *yaml.Node
	mapping := document.Content[0]
	for i := 0; i < len(mapping.Content); i += 2 {
		key, value := mapping.Content[i], mapping.Content[i+1]
		if key.Value != "applies_when" {
			continue
		}
		if appliesWhen != nil {
			return "", errors.New("duplicate applies_when frontmatter")
		}
		appliesWhen = value
	}
	if appliesWhen == nil {
		return "", errors.New("missing applies_when frontmatter")
	}
	if appliesWhen.Kind != yaml.ScalarNode || appliesWhen.Tag != "!!str" ||
		appliesWhen.Style == yaml.LiteralStyle || appliesWhen.Style == yaml.FoldedStyle ||
		strings.TrimSpace(appliesWhen.Value) == "" || strings.ContainsAny(appliesWhen.Value, "\r\n") {
		return "", errors.New("applies_when must be a nonblank single-line string value")
	}
	return appliesWhen.Value, nil
}
