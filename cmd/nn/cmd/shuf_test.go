package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShufCommandRegistered(t *testing.T) {
	state := &rootState{}
	cmd := newShufCmd(state)
	if cmd.Use != "shuf [<path>...]" {
		t.Fatalf("expected Use='shuf [<path>...]', got %q", cmd.Use)
	}
}

func TestShufStdinLines(t *testing.T) {
	input := "alpha\nbeta\ngamma\ndelta\nepsilon\n"
	var out bytes.Buffer
	err := runShuf(strings.NewReader(input), nil, &out, nil, 3, "lines")
	if err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "---") {
		t.Fatalf("expected separator '---' in output, got:\n%s", got)
	}
}

func TestShufStdinParagraphs(t *testing.T) {
	input := "paragraph one\nstill one\n\nparagraph two\n\nparagraph three\n"
	var out bytes.Buffer
	err := runShuf(strings.NewReader(input), nil, &out, nil, 2, "paragraphs")
	if err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "---") {
		t.Fatalf("expected separator '---' in output, got:\n%s", got)
	}
}

func TestShufFilePaths(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "sample.txt")
	content := "line one\nline two\nline three\nline four\nline five\n"
	if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := runShuf(nil, []string{f}, &out, nil, 2, "lines")
	if err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "---") {
		t.Fatalf("expected '---' separator, got:\n%s", got)
	}
}

func TestShufMultiplePaths(t *testing.T) {
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.txt")
	f2 := filepath.Join(dir, "b.txt")
	_ = os.WriteFile(f1, []byte("aaa\nbbb\nccc\n"), 0o644)
	_ = os.WriteFile(f2, []byte("ddd\neee\nfff\n"), 0o644)
	var out bytes.Buffer
	err := runShuf(nil, []string{f1, f2}, &out, nil, 3, "lines")
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() == 0 {
		t.Fatal("expected non-empty output")
	}
}

func TestShufCountCapsOutput(t *testing.T) {
	input := strings.Repeat("line\n", 100)
	var out bytes.Buffer
	err := runShuf(strings.NewReader(input), nil, &out, nil, 3, "lines")
	if err != nil {
		t.Fatal(err)
	}
	count := strings.Count(out.String(), "---")
	if count != 3 {
		t.Fatalf("expected 3 '---' separators for count=3, got %d", count)
	}
}

func TestShufDirectoryRecurse(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("alpha\nbeta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.txt"), []byte("gamma\ndelta\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := runShuf(nil, []string{root}, &out, nil, 10, "lines")
	if err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if !strings.Contains(got, "alpha") && !strings.Contains(got, "beta") {
		t.Fatalf("expected content from root file (alpha/beta), got:\n%s", got)
	}
	if !strings.Contains(got, "gamma") && !strings.Contains(got, "delta") {
		t.Fatalf("expected content from subdir file (gamma/delta), got:\n%s", got)
	}
}

func TestShufDirectoryOnlyRegularFiles(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	err := runShuf(nil, []string{root}, &out, nil, 10, "lines")
	if err != nil {
		t.Fatalf("expected no error (dir entry should be skipped), got: %v", err)
	}
}

func TestShufEmptyInput(t *testing.T) {
	var out bytes.Buffer
	err := runShuf(strings.NewReader(""), nil, &out, nil, 3, "lines")
	if err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Fatalf("expected empty output for empty input, got %q", out.String())
	}
}
