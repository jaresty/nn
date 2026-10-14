package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

const listPageTestLimit = 48_000

type listPageTestEnvelope struct {
	Snapshot      string            `json:"snapshot"`
	Page          int               `json:"page"`
	Pages         int               `json:"pages"`
	NextPage      int               `json:"next_page"`
	ResultCount   int               `json:"result_count"`
	TotalMatching int               `json:"total_matching"`
	Results       []json.RawMessage `json:"results"`
}

func TestListPaginationIsCompleteBoundedAndSnapshotStable(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	for i := range 1800 {
		n := newTestNoteForCLI(note.GenerateID(), fmt.Sprintf("page-note-%03d-%s", i, strings.Repeat("x", 80)), note.TypeObservation)
		writeNoteFile(t, nbDir, n)
	}

	plain, err := execute("list", "--json", "--fields", "id")
	if err != nil {
		t.Fatal(err)
	}
	var want []map[string]any
	if err := json.Unmarshal([]byte(plain), &want); err != nil {
		t.Fatal(err)
	}

	var got []map[string]any
	page, snapshot := 1, ""
	for {
		args := []string{"list", "--json", "--envelope", "--fields", "id", "--page", fmt.Sprint(page)}
		if page > 1 {
			args = append(args, "--snapshot", snapshot)
		}
		out, err := execute(args...)
		if err != nil {
			t.Fatalf("page %d: %v", page, err)
		}
		if len(out) > listPageTestLimit {
			t.Fatalf("page %d bytes = %d", page, len(out))
		}
		var env listPageTestEnvelope
		if err := json.Unmarshal([]byte(out), &env); err != nil {
			t.Fatal(err)
		}
		if page == 1 {
			snapshot = env.Snapshot
			if snapshot == "" || env.Pages < 2 {
				t.Fatalf("page 1 = %#v", env)
			}
		} else if env.Snapshot != snapshot {
			t.Fatalf("snapshot changed")
		}
		for _, raw := range env.Results {
			var value map[string]any
			if err := json.Unmarshal(raw, &value); err != nil {
				t.Fatal(err)
			}
			got = append(got, value)
		}
		if env.NextPage == 0 {
			break
		}
		page = env.NextPage
	}
	if len(got) != len(want) {
		t.Fatalf("reconstructed %d results, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i]["id"] != want[i]["id"] {
			t.Fatalf("result %d id = %v, want %v", i, got[i]["id"], want[i]["id"])
		}
	}

	if _, err := execute("list", "--json", "--envelope", "--fields", "id", "--page", "2"); err == nil {
		t.Fatal("page 2 without snapshot succeeded")
	}
	n := newTestNoteForCLI(note.GenerateID(), "snapshot mutation", note.TypeObservation)
	writeNoteFile(t, nbDir, n)
	if _, err := execute("list", "--json", "--envelope", "--fields", "id", "--page", "2", "--snapshot", snapshot); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale continuation error = %v", err)
	}
}

func TestListPaginationRejectsIncompatibleModes(t *testing.T) {
	_, execute := setupNotebook(t)
	cases := [][]string{
		{"list", "--page", "1"},
		{"list", "--json", "--page", "1"},
		{"list", "--json", "--envelope", "--snapshot", "abc"},
		{"list", "--json", "--envelope", "--page", "1", "--limit", "2"},
		{"list", "--json", "--envelope", "--page", "1", "--show-first"},
	}
	for _, args := range cases {
		if _, err := execute(args...); err == nil {
			t.Fatalf("%v succeeded", args)
		}
	}
}
