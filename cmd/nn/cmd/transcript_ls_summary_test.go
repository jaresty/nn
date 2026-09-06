package cmd

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"
)

func TestTranscriptLsSummaryLegacyText(t *testing.T) {
	dir := t.TempDir()
	writeSDKCLIFixture(t, dir)
	_, execute := setupNotebook(t)
	encoded, err := execute("transcript", "ls", dir, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []sessionRow
	if err := json.Unmarshal([]byte(encoded), &rows); err != nil || len(rows) != 1 {
		t.Fatalf("rows: %s %v", encoded, err)
	}
	r := rows[0]
	want := fmt.Sprintf("%s  %-11s  %2d agents  cost=%-7d  %s  %s\n", r.Modified[:10], r.Schema, r.AgentCount, r.TotalCost, r.Session, r.TreePreview)
	out, err := execute("transcript", "ls", dir)
	if err != nil || out != want {
		t.Fatalf("ASSERT_SUMMARY_LEGACY_TEXT: got %q want %q error=%v", out, want, err)
	}
}

func TestTranscriptLsSummaryCLI(t *testing.T) {
	for _, schema := range []string{"sdk", "pi", "unknown"} {
		t.Run(schema, func(t *testing.T) {
			dir := t.TempDir()
			switch schema {
			case "sdk":
				writeSDKCLIFixture(t, dir)
			case "pi":
				writePiFixture(t, dir)
			default:
				writeTranscriptFile(t, filepath.Join(dir, "unknown.jsonl"), "{}\n")
			}
			_, execute := setupNotebook(t)
			out, err := execute("transcript", "ls", dir, "--json")
			if err != nil {
				t.Fatal(err)
			}
			var rows []map[string]json.RawMessage
			if err := json.Unmarshal([]byte(out), &rows); err != nil || len(rows) != 1 {
				t.Fatalf("rows: %s %v", out, err)
			}
			raw, exists := rows[0]["summary"]
			if !exists {
				t.Fatal("ASSERT_LS_SUMMARY_PRESENT: missing summary")
			}
			if schema == "unknown" {
				if string(raw) != "null" {
					t.Fatalf("ASSERT_LS_SUMMARY_UNAVAILABLE: %s", raw)
				}
				return
			}
			var s struct {
				Cost struct {
					Status      string
					Total       int `json:"total_tokens"`
					Measured    int `json:"measured_agents"`
					Unavailable int `json:"unavailable_agents"`
					Input       int `json:"input_tokens"`
					Output      int `json:"output_tokens"`
				}
				Topology struct {
					Roots    int `json:"root_count"`
					Edges    int `json:"edge_count"`
					Depth    int `json:"max_depth"`
					Children int `json:"max_children"`
				}
				TopologyStatus string `json:"topology_status"`
				Types          []struct {
					Type  string
					Count int
				} `json:"agent_types"`
			}
			if err := json.Unmarshal(raw, &s); err != nil {
				t.Fatal(err)
			}
			if schema == "sdk" {
				if s.Cost.Status != "complete" || s.Cost.Total != 90 || s.Cost.Measured != 3 || s.Cost.Unavailable != 0 || s.Cost.Input != 60 || s.Cost.Output != 30 {
					t.Fatalf("ASSERT_LS_SUMMARY_COST: %+v", s.Cost)
				}
				if s.TopologyStatus != "complete" || s.Topology.Roots != 1 || s.Topology.Edges != 2 || s.Topology.Depth != 2 || s.Topology.Children != 1 {
					t.Fatalf("ASSERT_LS_SUMMARY_TOPOLOGY: %+v", s)
				}
				if len(s.Types) != 2 || s.Types[0].Type != "general-purpose" || s.Types[0].Count != 2 {
					t.Fatalf("ASSERT_LS_SUMMARY_TYPES: %+v", s.Types)
				}
			} else {
				if s.Cost.Status != "partial" || s.Cost.Total != 15 || s.Cost.Measured != 1 || s.Cost.Unavailable != 1 {
					t.Fatalf("ASSERT_LS_SUMMARY_COST: %+v", s.Cost)
				}
			}
			var legacyCost int
			json.Unmarshal(rows[0]["total_cost"], &legacyCost)
			if legacyCost != s.Cost.Total {
				t.Fatal("ASSERT_LS_SUMMARY_LEGACY_COST: mismatch")
			}
		})
	}
}
