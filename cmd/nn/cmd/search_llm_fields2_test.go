package cmd

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// D4: is_protocol boolean — true for protocol notes or notes with governs links.
func TestSearchJSONIsProtocolTrue(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	p := newTestNoteForCLI(note.GenerateID(), "Capture Protocol", note.TypeProtocol)
	p.Body = "Protocol for capturing notes."
	writeNoteFile(t, nbDir, p)

	out, err := execute("list", "--search", "capturing", "--json")
	if err != nil {
		t.Fatalf("nn list --search --json: %v", err)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(results) == 0 {
		t.Fatal("no results returned")
	}
	val, ok := results[0]["is_protocol"]
	if !ok {
		t.Fatalf("'is_protocol' field missing from search JSON result: %v", results[0])
	}
	if val != true {
		t.Errorf("'is_protocol' = %v, want true for protocol-type note", val)
	}
}

func TestSearchJSONIsProtocolFalse(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	c := newTestNoteForCLI(note.GenerateID(), "Concept About Caching", note.TypeConcept)
	c.Body = "Describes cache eviction."
	writeNoteFile(t, nbDir, c)

	out, err := execute("list", "--search", "cache", "--json")
	if err != nil {
		t.Fatalf("nn list --search --json: %v", err)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(results) == 0 {
		t.Fatal("no results returned")
	}
	val, ok := results[0]["is_protocol"]
	if !ok {
		t.Fatalf("'is_protocol' field missing from search JSON result: %v", results[0])
	}
	if val != false {
		t.Errorf("'is_protocol' = %v, want false for concept-type note", val)
	}
}

// D5a: link_count — equals len(n.Links) for a note with outgoing links.
func TestSearchJSONLinkCountValue(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	target := newTestNoteForCLI(note.GenerateID(), "Target Note", note.TypeConcept)
	target.Body = "A target."
	n := newTestNoteForCLI(note.GenerateID(), "Linker Note Outgoing", note.TypeConcept)
	n.Body = "Discusses outgoing links."
	n.Links = []note.Link{{TargetID: target.ID, Annotation: "relates to"}}
	writeNoteFile(t, nbDir, target)
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--search", "outgoing", "--json")
	if err != nil {
		t.Fatalf("nn list --search --json: %v", err)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(results) == 0 {
		t.Fatal("no results returned")
	}
	val, ok := results[0]["link_count"]
	if !ok {
		t.Fatalf("'link_count' field missing from search JSON result: %v", results[0])
	}
	count, isNum := val.(float64)
	if !isNum {
		t.Fatalf("'link_count' is not a number: %T %v", val, val)
	}
	if int(count) != 1 {
		t.Errorf("'link_count' = %d, want 1 (note has 1 outgoing link)", int(count))
	}
}

// D5b: backlink_count — equals count of notes linking to this note.
func TestSearchJSONBacklinkCountValue(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	// target is the note we'll search for; linker links to it.
	target := newTestNoteForCLI(note.GenerateID(), "Backlink Target Concept", note.TypeConcept)
	target.Body = "Discusses backlink target."
	linker := newTestNoteForCLI(note.GenerateID(), "Linker Note", note.TypeConcept)
	linker.Body = "Links to backlink target."
	linker.Links = []note.Link{{TargetID: target.ID, Annotation: "references"}}
	writeNoteFile(t, nbDir, target)
	writeNoteFile(t, nbDir, linker)

	out, err := execute("list", "--search", "backlink target", "--json")
	if err != nil {
		t.Fatalf("nn list --search --json: %v", err)
	}
	var results []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	// Find the target note in results.
	var targetResult map[string]interface{}
	for _, r := range results {
		if r["title"] == "Backlink Target Concept" {
			targetResult = r
			break
		}
	}
	if targetResult == nil {
		t.Fatal("target note not found in search results")
	}
	val, ok := targetResult["backlink_count"]
	if !ok {
		t.Fatalf("'backlink_count' field missing from search JSON result: %v", targetResult)
	}
	count, isNum := val.(float64)
	if !isNum {
		t.Fatalf("'backlink_count' is not a number: %T %v", val, val)
	}
	if int(count) != 1 {
		t.Errorf("'backlink_count' = %d, want 1 (one note links to target)", int(count))
	}
}

// D6: envelope output — query, result_count, total_matching, results when --envelope is passed.
func TestSearchJSONEnvelope(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Envelope Test Note", note.TypeConcept)
	n.Body = "Discusses envelope wrapping."
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--search", "envelope", "--json", "--envelope")
	if err != nil {
		t.Fatalf("nn list --search --json --envelope: %v", err)
	}
	var env map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &env); err != nil {
		t.Fatalf("output is not a JSON object (expected envelope): %v\n%s", err, out)
	}
	if _, ok := env["query"]; !ok {
		t.Errorf("envelope missing 'query' field: %v", env)
	}
	if _, ok := env["result_count"]; !ok {
		t.Errorf("envelope missing 'result_count' field: %v", env)
	}
	if _, ok := env["total_matching"]; !ok {
		t.Errorf("envelope missing 'total_matching' field: %v", env)
	}
	if _, ok := env["results"]; !ok {
		t.Errorf("envelope missing 'results' field: %v", env)
	}
}

// D6 inverse: without --envelope, output remains a plain array (no regression).
func TestSearchJSONNoEnvelopeByDefault(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI(note.GenerateID(), "Plain Array Note", note.TypeConcept)
	n.Body = "Discusses plain array output."
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--search", "plain", "--json")
	if err != nil {
		t.Fatalf("nn list --search --json: %v", err)
	}
	// Output must be a JSON array, not an object.
	trimmed := strings.TrimSpace(out)
	if !strings.HasPrefix(trimmed, "[") {
		t.Errorf("default output should be a JSON array, got: %q", trimmed[:min(len(trimmed), 40)])
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
