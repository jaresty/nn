package gitlocal

import (
	"os"
	"path/filepath"
	"testing"
)

// property [32a]: an exclusive create must not overwrite an existing file — it
// returns an error so the caller can retry with a fresh ID rather than silently
// clobbering another note written concurrently at the same path.
func TestExclusiveWriteFileFailsOnExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "note.md")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}

	err := exclusiveWriteFile(path, []byte("intruder"))
	if err == nil {
		t.Fatalf("exclusiveWriteFile overwrote an existing file, want error")
	}
	if !os.IsExist(err) {
		t.Fatalf("err = %v, want an IsExist error", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "original" {
		t.Fatalf("existing file content = %q, want %q (must be preserved)", got, "original")
	}
}

// exclusiveWriteFile must succeed when the path is free.
func TestExclusiveWriteFileWritesWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fresh.md")
	if err := exclusiveWriteFile(path, []byte("hello")); err != nil {
		t.Fatalf("exclusiveWriteFile on absent path: %v", err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "hello" {
		t.Fatalf("content = %q, want %q", got, "hello")
	}
}
