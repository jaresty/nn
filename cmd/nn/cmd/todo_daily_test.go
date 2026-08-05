package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// property [1a]: non-daily note with old Created must NOT be excluded
func TestTodoList_Property1a_NonDailyNotSkipped(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	yesterday := time.Now().UTC().AddDate(0, 0, -1).Truncate(time.Second)
	n := newTestNoteForCLI(note.GenerateID(), "Regular old note", note.TypeObservation)
	n.Created = yesterday
	n.Modified = yesterday
	n.Body = "- [ ] do something"
	writeNoteFile(t, nbDir, n)

	out, err := execute("todo", "list")
	if err != nil {
		t.Fatalf("todo list: %v", err)
	}
	if !strings.Contains(out, "do something") {
		t.Errorf("expected non-daily old note in todo list, got:\n%s", out)
	}
}

// property [1b]: daily note created TODAY must NOT be excluded
func TestTodoList_Property1b_TodaysDailyNotSkipped(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	now := time.Now().UTC().Truncate(time.Second)
	n := newTestNoteForCLI(note.GenerateID(), "Daily: today", note.TypeObservation)
	n.Tags = []string{"daily"}
	n.Created = now
	n.Modified = now
	n.Body = "- [ ] do something today"
	writeNoteFile(t, nbDir, n)

	out, err := execute("todo", "list")
	if err != nil {
		t.Fatalf("todo list: %v", err)
	}
	if !strings.Contains(out, "do something today") {
		t.Errorf("expected today's daily note in todo list, got:\n%s", out)
	}
}

// property [2]: historical daily note excluded unconditionally — default and --all
func TestTodoList_Property2_HistoricalDailyExcluded(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	yesterday := time.Now().UTC().AddDate(0, 0, -1).Truncate(time.Second)
	n := newTestNoteForCLI(note.GenerateID(), "Daily: yesterday", note.TypeObservation)
	n.Tags = []string{"daily"}
	n.Created = yesterday
	n.Modified = yesterday
	n.Body = "- [ ] stale todo"
	writeNoteFile(t, nbDir, n)

	out, err := execute("todo", "list")
	if err != nil {
		t.Fatalf("todo list: %v", err)
	}
	if strings.Contains(out, "stale todo") {
		t.Errorf("expected historical daily note excluded by default, got:\n%s", out)
	}

	out, err = execute("todo", "list", "--all")
	if err != nil {
		t.Fatalf("todo list --all: %v", err)
	}
	if strings.Contains(out, "stale todo") {
		t.Errorf("expected historical daily note excluded with --all, got:\n%s", out)
	}
}
