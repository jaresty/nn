package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

type graphShowPageResult struct {
	Snapshot string `json:"snapshot"`
	Page     int    `json:"page"`
	Pages    int    `json:"pages"`
	NextPage int    `json:"next_page"`
	Center   string `json:"center"`
	Nodes    []struct {
		ID string `json:"id"`
	} `json:"nodes"`
	Edges []struct {
		From string `json:"from"`
		To   string `json:"to"`
	} `json:"edges"`
}

func TestGraphShowJSONPaginationIsBoundedAndLossless(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root := newTestNoteForCLI("20990101000000-0000", "root", note.TypeModel)
	for i := 1; i <= 200; i++ {
		id := fmt.Sprintf("20990101000000-%04d", i)
		n := newTestNoteForCLI(id, fmt.Sprintf("topology node %03d", i), note.TypeConcept)
		n.Links = []note.Link{{TargetID: root.ID, Type: "extends", Annotation: strings.Repeat("annotation ", 80)}}
		writeNoteFile(t, nbDir, n)
	}
	writeNoteFile(t, nbDir, root)

	firstOut, err := execute("graph", "show", "--focus", root.ID, "--depth", "1", "--direction", "both", "--format", "json", "--page", "1")
	if err != nil {
		t.Fatalf("page 1: %v", err)
	}
	if len(firstOut) > graphShowPageMaxBytes {
		t.Fatalf("page 1 encoded size = %d, limit %d", len(firstOut), graphShowPageMaxBytes)
	}
	var first graphShowPageResult
	if err := json.Unmarshal([]byte(firstOut), &first); err != nil {
		t.Fatalf("page 1 JSON: %v", err)
	}
	if first.Pages < 2 || first.NextPage != 2 || len(first.Snapshot) != 64 {
		t.Fatalf("page 1 metadata = %+v", first)
	}

	nodeIDs := map[string]bool{}
	edgeCount := 0
	for page := 1; page <= first.Pages; page++ {
		out := firstOut
		if page > 1 {
			out, err = execute("graph", "show", "--focus", root.ID, "--depth", "1", "--direction", "both", "--format", "json", "--page", strconv.Itoa(page), "--snapshot", first.Snapshot)
			if err != nil {
				t.Fatalf("page %d: %v", page, err)
			}
		}
		if len(out) > graphShowPageMaxBytes {
			t.Fatalf("page %d encoded size = %d, limit %d", page, len(out), graphShowPageMaxBytes)
		}
		var got graphShowPageResult
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("page %d JSON: %v", page, err)
		}
		for _, n := range got.Nodes {
			nodeIDs[n.ID] = true
		}
		edgeCount += len(got.Edges)
	}
	if len(nodeIDs) != 201 || edgeCount != 200 {
		t.Fatalf("reconstructed topology = %d nodes, %d edges; want 201, 200", len(nodeIDs), edgeCount)
	}
}

func TestGraphShowPaginationValidation(t *testing.T) {
	_, execute := setupNotebook(t)
	cases := [][]string{
		{"graph", "show", "--page", "1"},
		{"graph", "show", "--format", "json", "--page", "2"},
		{"graph", "show", "--format", "json", "--snapshot", strings.Repeat("a", 64)},
	}
	for _, args := range cases {
		if _, err := execute(args...); err == nil {
			t.Fatalf("expected rejection for %v", args)
		}
	}
}
