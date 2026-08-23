package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

type presentationBudgetTest struct {
	Tier    string   `json:"tier"`
	Length  string   `json:"length"`
	Include []string `json:"include"`
	Note    string   `json:"note"`
}

func TestGraphShowPresentationHintsPreserveBodiesAndLegacyOutput(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	leaf := newTestNoteForCLI("20260101000000-0001", "Leaf", note.TypeConcept)
	connected := newTestNoteForCLI("20260101000000-0002", "Connected", note.TypeConcept)
	hub := newTestNoteForCLI("20260101000000-0003", "Hub", note.TypeConcept)
	indexHub := newTestNoteForCLI("20260101000000-0004", "Index hub", note.TypeConcept)
	leaf.Body = "Complete leaf body; do not truncate."
	connected.Body = "Complete connected body; do not truncate."
	hub.Body = "Complete hub body; do not truncate."
	indexHub.Body = "Complete index body; do not truncate."
	indexHub.Tags = []string{"index"}
	nodes := []*note.Note{leaf, connected, hub, indexHub}
	for i := 0; i < 5; i++ {
		source := newTestNoteForCLI("20260101000000-01"+string(rune('0'+i))+"0", "Source", note.TypeConcept)
		if i == 0 {
			source.Links = append(source.Links, note.Link{TargetID: leaf.ID})
		}
		if i < 2 {
			source.Links = append(source.Links, note.Link{TargetID: connected.ID})
		}
		source.Links = append(source.Links, note.Link{TargetID: hub.ID}, note.Link{TargetID: indexHub.ID})
		nodes = append(nodes, source)
	}
	for _, n := range nodes {
		writeNoteFile(t, nbDir, n)
	}

	legacy, err := execute("graph", "show", "--format", "json", "--bodies")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(legacy, "summary_budget") {
		t.Fatalf("legacy JSON contains presentation hints: %s", legacy)
	}
	var legacyResult struct {
		Nodes []struct {
			ID   string `json:"id"`
			Body string `json:"body"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(legacy), &legacyResult); err != nil {
		t.Fatal(err)
	}
	legacyBodyByID := make(map[string]string, len(legacyResult.Nodes))
	for _, n := range legacyResult.Nodes {
		legacyBodyByID[n.ID] = n.Body
	}
	out, err := execute("graph", "show", "--format", "json", "--bodies", "--presentation-hints")
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Nodes []struct {
			ID            string                 `json:"id"`
			Body          string                 `json:"body"`
			SummaryBudget presentationBudgetTest `json:"summary_budget"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("hint JSON: %v\n%s", err, out)
	}
	byID := make(map[string]struct {
		Body   string
		Budget presentationBudgetTest
	})
	for _, n := range result.Nodes {
		byID[n.ID] = struct {
			Body   string
			Budget presentationBudgetTest
		}{n.Body, n.SummaryBudget}
	}
	for id, want := range map[string]struct {
		tier, length, note string
	}{
		leaf.ID:      {"leaf", "one clause", ""},
		connected.ID: {"connected", "one sentence", ""},
		hub.ID:       {"hub", "2–3 sentences", ""},
		indexHub.ID:  {"connected", "one sentence", "aggregation hub: summarize as connected"},
	} {
		got := byID[id]
		if got.Body != legacyBodyByID[id] {
			t.Errorf("node %s body changed by presentation hints: got %q, baseline %q", id, got.Body, legacyBodyByID[id])
		}
		if got.Budget.Tier != want.tier || got.Budget.Length != want.length || got.Budget.Note != want.note {
			t.Errorf("node %s budget = %#v, want tier=%q length=%q note=%q", id, got.Budget, want.tier, want.length, want.note)
		}
		if len(got.Budget.Include) == 0 {
			t.Errorf("node %s budget has no include guidance", id)
		}
	}

	text, err := execute("graph", "show", "--format", "text", "--presentation-hints")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "relay budget: 2–3 sentences") || !strings.Contains(text, "aggregation hub: summarize as connected") {
		t.Fatalf("text output lacks presentation hints:\n%s", text)
	}
	legacyText, err := execute("graph", "show", "--format", "text")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(legacyText, "relay budget:") {
		t.Fatalf("legacy text contains presentation hints:\n%s", legacyText)
	}
}
