package note_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestDeadSymbolsRemovedFromProduction asserts that BM25RRF, GetOrComputeIDF,
// and GetOrComputeIDFPath do not appear in non-test production Go files.
func TestDeadSymbolsRemovedFromProduction(t *testing.T) {
	deadSymbols := regexp.MustCompile(`\b(BM25RRF|GetOrComputeIDF|GetOrComputeIDFPath)\b`)

	roots := []string{
		filepath.Join("..", "..", ".."),
	}

	for _, root := range roots {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() && (d.Name() == ".git" || d.Name() == "vendor") {
				return filepath.SkipDir
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			if strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			for _, line := range strings.Split(string(data), "\n") {
				if deadSymbols.MatchString(line) {
					t.Errorf("dead symbol found in production file %s: %s", path, strings.TrimSpace(line))
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("WalkDir: %v", err)
		}
	}
}
