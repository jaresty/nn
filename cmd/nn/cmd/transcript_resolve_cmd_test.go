package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranscriptResolveCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".pi", "agent", "sessions", "project", "session-123.jsonl")
	writeTranscriptFile(t, path, `{"type":"session","version":3,"id":"session-123"}`+"\n")
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err = filepath.Abs(canonical)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("text emits canonical path", func(t *testing.T) {
		_, execute := setupNotebook(t)
		out, err := execute("transcript", "resolve", "session-123")
		if err != nil {
			t.Fatal(err)
		}
		if strings.TrimSpace(out) != canonical {
			t.Fatalf("path=%q want=%q", strings.TrimSpace(out), canonical)
		}
	})

	t.Run("json emits identity and path", func(t *testing.T) {
		_, execute := setupNotebook(t)
		out, err := execute("transcript", "resolve", "session-123", "--json")
		if err != nil {
			t.Fatal(err)
		}
		var got transcriptResolveResult
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatal(err)
		}
		want := (transcriptResolveResult{Session: "session-123", Path: canonical})
		if got != want {
			t.Fatalf("result=%+v want=%+v", got, want)
		}
	})
}

func TestTranscriptResolveCommandPropagatesLookupErrors(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeTranscriptFile(t, filepath.Join(home, ".pi", "agent", "sessions", "project", "other.jsonl"), `{"type":"session","version":3,"id":"other"}`+"\n")

	_, execute := setupNotebook(t)
	_, err := execute("transcript", "resolve", "missing", "--json")
	if err == nil || !strings.Contains(err.Error(), `transcript session "missing" not found`) {
		t.Fatalf("expected resolver diagnostic, got %v", err)
	}
}
