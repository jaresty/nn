package cmd

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Explicit inputs are strict; recursive discovery reports unsupported entries.
// Directory symlinks inside a walk are not followed (avoids cycles and escapes).
func transcriptSearchInputs(inputs []string) ([]string, int, error) {
	seen := map[string]string{}
	visited := map[string]bool{}
	skipped := 0
	add := func(path string, explicit bool) error {
		canonical, err := filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		canonical, err = filepath.Abs(canonical)
		if err != nil {
			return err
		}
		if visited[canonical] {
			if explicit && seen[canonical] == "" {
				return fmt.Errorf("search %s: unsupported transcript", path)
			}
			return nil
		}
		visited[canonical] = true
		info, err := os.Stat(canonical)
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("search %s: not a regular file", path)
		}
		// Classification alone cannot distinguish unreadable from unsupported files.
		f, err := os.Open(canonical)
		if err != nil {
			return err
		}
		if err = f.Close(); err != nil {
			return err
		}
		if classifyTranscript(canonical) == schemaUnknown {
			if explicit {
				return fmt.Errorf("search %s: unsupported transcript", path)
			}
			skipped++
			return nil
		}
		seen[canonical] = path
		return nil
	}
	for _, input := range inputs {
		root := input
		info, err := os.Stat(root)
		if err != nil {
			return nil, skipped, err
		}
		if !info.IsDir() {
			if err = add(root, true); err != nil {
				return nil, skipped, err
			}
			continue
		}
		entry, err := os.Lstat(root)
		if err != nil {
			return nil, skipped, err
		}
		if entry.Mode()&os.ModeSymlink != 0 {
			root, err = filepath.EvalSymlinks(root)
			if err != nil {
				return nil, skipped, err
			}
		}
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() {
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 {
				target, err := os.Stat(path)
				if err != nil {
					return err
				}
				if target.IsDir() {
					skipped++
					return nil
				}
			}
			if !strings.HasSuffix(path, ".jsonl") && !strings.HasSuffix(path, ".output") {
				skipped++
				return nil
			}
			return add(path, false)
		})
		if err != nil {
			return nil, skipped, err
		}
	}
	files := make([]string, 0, len(seen))
	for path := range seen {
		files = append(files, path)
	}
	sort.Strings(files)
	for i, canonical := range files {
		files[i] = seen[canonical]
	}
	return files, skipped, nil
}
