package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Opt-in benchmark fixture creation. Only authenticated parent locators are
// copied; unresolved locators point to absent files inside the frozen directory.
func TestFreezeObservationBenchmark(t *testing.T) {
	source, dest := os.Getenv("NN_FREEZE_SOURCE"), os.Getenv("NN_FREEZE_DEST")
	if source == "" || dest == "" {
		t.Skip("set explicit NN_FREEZE_SOURCE and NN_FREEZE_DEST")
	}
	entries, err := os.ReadDir(dest)
	if err != nil || len(entries) != 0 {
		t.Fatal("destination must exist and be empty", err)
	}
	capture, err := newTranscriptCapture(source)
	if err != nil {
		t.Fatal(err)
	}
	root := bytes.Clone(capture.Sources[capture.Path].Raw)
	count := 0
	for i, loc := range piBackgroundLocators(capture.Sources[capture.Path].Records) {
		name := loc.AgentID
		if strings.ContainsAny(name, "/\\") || name == "." || name == ".." {
			name = captureHash([]byte(name))
		}
		target := filepath.Join(dest, "pi-subagents-frozen", fmt.Sprint(i), "tasks", name+".output")
		if safe := capture.resolve(loc.Path, loc.AgentID); safe != "" {
			if err = os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(target, capture.Sources[safe].Raw, 0600); err != nil {
				t.Fatal(err)
			}
			info, e := os.Stat(safe)
			if e != nil {
				t.Fatal(e)
			}
			if e = os.Chtimes(target, info.ModTime(), info.ModTime()); e != nil {
				t.Fatal(e)
			}
			count++
		}
		from, _ := json.Marshal(loc.Path)
		to, _ := json.Marshal(target)
		root = bytes.ReplaceAll(root, from[1:len(from)-1], to[1:len(to)-1])
	}
	path := filepath.Join(dest, "session.jsonl")
	if err = os.WriteFile(path, root, 0600); err != nil {
		t.Fatal(err)
	}
	records, err := readRecords(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, loc := range piBackgroundLocators(records) {
		rel, e := filepath.Rel(dest, loc.Path)
		if e != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			t.Fatal("unfrozen locator remains", loc.Path, e)
		}
	}
	t.Logf("FROZEN_OBSERVATION path=%s copied=%d parent_sha256=%s", path, count, captureHash(root))
}
