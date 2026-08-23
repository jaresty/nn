package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

type bridgeSearchResult struct {
	ID             string  `json:"id"`
	Title          string  `json:"title"`
	Score          int     `json:"score"`
	RelevanceScore float64 `json:"relevance_score"`
}

func TestGraphBridgesSearchProjectsOntoFullGraphBridges(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	left := newTestNoteForCLI("20260101000000-0001", "Left context", note.TypeConcept)
	matchingBridge := newTestNoteForCLI("20260101000000-0002", "Quasarbridge crossing", note.TypeConcept)
	right := newTestNoteForCLI("20260101000000-0003", "Right context", note.TypeConcept)
	left.Links = []note.Link{{TargetID: matchingBridge.ID}}
	matchingBridge.Links = []note.Link{{TargetID: right.ID}}

	matchingNonBridge := newTestNoteForCLI("20260101000000-0004", "Quasarbridge isolated", note.TypeConcept)
	otherLeft := newTestNoteForCLI("20260101000000-0005", "Other left", note.TypeConcept)
	unrelatedBridge := newTestNoteForCLI("20260101000000-0006", "Unrelated crossing", note.TypeConcept)
	otherRight := newTestNoteForCLI("20260101000000-0007", "Other right", note.TypeConcept)
	otherLeft.Links = []note.Link{{TargetID: unrelatedBridge.ID}}
	unrelatedBridge.Links = []note.Link{{TargetID: otherRight.ID}}

	for _, n := range []*note.Note{left, matchingBridge, right, matchingNonBridge, otherLeft, unrelatedBridge, otherRight} {
		writeNoteFile(t, nbDir, n)
	}

	out, err := execute("graph", "bridges", "--search", "quasarbridge", "--format", "json")
	if err != nil {
		t.Fatalf("graph bridges --search: %v", err)
	}
	var got []bridgeSearchResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("graph bridges --search JSON: %v\n%s", err, out)
	}
	if len(got) != 1 || got[0].ID != matchingBridge.ID {
		t.Fatalf("projected bridges = %#v, want only full-graph bridge %s", got, matchingBridge.ID)
	}
	if got[0].Score <= 0 || got[0].RelevanceScore <= 0 {
		t.Fatalf("bridge result lacks structural and relevance scores: %#v", got[0])
	}
}

func TestGraphBridgesSearchRanksByRelevanceBeforeBridgeScore(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	highRel := newTestNoteForCLI("20260101000000-0010", "Needle needle crossing", note.TypeConcept)
	lowRelHighBridge := newTestNoteForCLI("20260101000000-0020", "Needle crossing", note.TypeConcept)
	left1 := newTestNoteForCLI("20260101000000-0011", "L1", note.TypeConcept)
	right1 := newTestNoteForCLI("20260101000000-0012", "R1", note.TypeConcept)
	left1.Links = []note.Link{{TargetID: highRel.ID}}
	highRel.Links = []note.Link{{TargetID: right1.ID}}

	left2a := newTestNoteForCLI("20260101000000-0021", "L2a", note.TypeConcept)
	left2b := newTestNoteForCLI("20260101000000-0022", "L2b", note.TypeConcept)
	right2a := newTestNoteForCLI("20260101000000-0023", "R2a", note.TypeConcept)
	right2b := newTestNoteForCLI("20260101000000-0024", "R2b", note.TypeConcept)
	left2a.Links = []note.Link{{TargetID: lowRelHighBridge.ID}}
	left2b.Links = []note.Link{{TargetID: lowRelHighBridge.ID}}
	lowRelHighBridge.Links = []note.Link{{TargetID: right2a.ID}, {TargetID: right2b.ID}}

	for _, n := range []*note.Note{highRel, lowRelHighBridge, left1, right1, left2a, left2b, right2a, right2b} {
		writeNoteFile(t, nbDir, n)
	}
	out, err := execute("graph", "bridges", "--search", "needle", "--format", "json")
	if err != nil {
		t.Fatal(err)
	}
	var got []bridgeSearchResult
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("bridge count = %d, want 2: %s", len(got), out)
	}
	if got[0].ID != highRel.ID || got[0].RelevanceScore <= got[1].RelevanceScore || got[0].Score >= got[1].Score {
		t.Fatalf("bridges not relevance-first: %#v", got)
	}
}

func TestGraphBridgesSearchRequiresJSONAndNonBlankQuery(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("graph", "bridges", "--search", "needle")
	if err == nil || !strings.Contains(err.Error(), "--search requires --format json") {
		t.Fatalf("text bridge search error = %v", err)
	}
	for _, query := range []string{"", " \t "} {
		_, err := execute("graph", "bridges", "--search", query, "--format", "json")
		if err == nil || !strings.Contains(err.Error(), "--search requires a non-blank query") {
			t.Errorf("bridge search %q error = %v", query, err)
		}
	}
}

func TestGraphBridgesSearchIsDocumentedForScan(t *testing.T) {
	_, execute := setupNotebook(t)
	help, err := execute("graph", "bridges", "--help")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(help, "--search") {
		t.Fatalf("bridges help missing --search:\n%s", help)
	}
	guide, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"nn graph bridges --search \"<query>\" --format json", "Peek", "Recenter"} {
		if !strings.Contains(string(guide), required) {
			t.Errorf("nn-guide missing %q", required)
		}
	}
}
