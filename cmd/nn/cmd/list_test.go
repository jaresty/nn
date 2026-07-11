package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

func TestListAll(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept))
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Beta", note.TypeArgument))

	out, err := execute("list")
	if err != nil {
		t.Fatalf("nn list: %v", err)
	}
	if !strings.Contains(out, "Alpha") || !strings.Contains(out, "Beta") {
		t.Errorf("list output missing notes: %q", out)
	}
}

func TestListFilterByType(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Concept Note", note.TypeConcept))
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Argument Note", note.TypeArgument))

	out, err := execute("list", "--type", "concept")
	if err != nil {
		t.Fatalf("nn list --type concept: %v", err)
	}
	if !strings.Contains(out, "Concept Note") {
		t.Errorf("output missing 'Concept Note': %q", out)
	}
	if strings.Contains(out, "Argument Note") {
		t.Errorf("output should not contain 'Argument Note': %q", out)
	}
}

func TestListFilterByStatus(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	draft := newTestNoteForCLI(note.GenerateID(), "Draft Note", note.TypeConcept)
	reviewed := newTestNoteForCLI(note.GenerateID(), "Reviewed Note", note.TypeConcept)
	reviewed.Status = note.StatusReviewed
	writeNoteFile(t, nbDir, draft)
	writeNoteFile(t, nbDir, reviewed)

	out, err := execute("list", "--status", "reviewed")
	if err != nil {
		t.Fatalf("nn list --status reviewed: %v", err)
	}
	if !strings.Contains(out, "Reviewed Note") {
		t.Errorf("output missing 'Reviewed Note': %q", out)
	}
	if strings.Contains(out, "Draft Note") {
		t.Errorf("output should not contain 'Draft Note': %q", out)
	}
}

func TestListFilterByTag(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	tagged := newTestNoteForCLI(note.GenerateID(), "Tagged", note.TypeConcept)
	tagged.Tags = []string{"zettelkasten"}
	untagged := newTestNoteForCLI(note.GenerateID(), "Untagged", note.TypeConcept)
	writeNoteFile(t, nbDir, tagged)
	writeNoteFile(t, nbDir, untagged)

	out, err := execute("list", "--tag", "zettelkasten")
	if err != nil {
		t.Fatalf("nn list --tag: %v", err)
	}
	if !strings.Contains(out, "Tagged") {
		t.Errorf("output missing 'Tagged': %q", out)
	}
	if strings.Contains(out, "Untagged") {
		t.Errorf("output should not contain 'Untagged': %q", out)
	}
}

func TestListOrphan(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	orphan := newTestNoteForCLI(note.GenerateID(), "Orphan", note.TypeConcept)
	linked := newTestNoteForCLI(note.GenerateID(), "Linked", note.TypeConcept)
	target := newTestNoteForCLI(note.GenerateID(), "Target", note.TypeConcept)
	linked.Links = []note.Link{{TargetID: target.ID, Annotation: "relates to"}}
	writeNoteFile(t, nbDir, orphan)
	writeNoteFile(t, nbDir, linked)
	writeNoteFile(t, nbDir, target)

	out, err := execute("list", "--orphan")
	if err != nil {
		t.Fatalf("nn list --orphan: %v", err)
	}
	if !strings.Contains(out, "Orphan") {
		t.Errorf("output missing 'Orphan': %q", out)
	}
	if strings.Contains(out, "Linked") || strings.Contains(out, "Target") {
		t.Errorf("output should not contain linked notes: %q", out)
	}
}

func TestListJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "JSON Note", note.TypeConcept))

	out, err := execute("list", "--json")
	if err != nil {
		t.Fatalf("nn list --json: %v", err)
	}
	var result []map[string]any
	mustJSON(t, out, &result)
	if len(result) != 1 {
		t.Errorf("JSON list count = %d, want 1", len(result))
	}
}

func TestListLimit(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	for i := 0; i < 5; i++ {
		writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Note", note.TypeConcept))
	}
	out, err := execute("list", "--limit", "2", "--json")
	if err != nil {
		t.Fatalf("nn list --limit: %v", err)
	}
	var result []map[string]any
	mustJSON(t, out, &result)
	if len(result) != 2 {
		t.Errorf("limited list count = %d, want 2", len(result))
	}
}

// Assertion: --global returns only protocols with no outgoing governs links.
func TestListGlobalProtocols(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	global := newTestNoteForCLI(note.GenerateID(), "Global Protocol", note.TypeProtocol)
	contextual := newTestNoteForCLI(note.GenerateID(), "Contextual Protocol", note.TypeProtocol)
	target := newTestNoteForCLI(note.GenerateID(), "Target Note", note.TypeConcept)
	contextual.Links = []note.Link{{TargetID: target.ID, Annotation: "governs", Type: "governs"}}
	writeNoteFile(t, nbDir, global)
	writeNoteFile(t, nbDir, contextual)
	writeNoteFile(t, nbDir, target)

	out, err := execute("list", "--global")
	if err != nil {
		t.Fatalf("nn list --global: %v", err)
	}
	if !strings.Contains(out, "Global Protocol") {
		t.Errorf("expected global protocol in output:\n%s", out)
	}
	if strings.Contains(out, "Contextual Protocol") {
		t.Errorf("expected contextual protocol excluded from output:\n%s", out)
	}
}

// Assertion: --global excludes non-protocol notes.
func TestListGlobalExcludesNonProtocol(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	proto := newTestNoteForCLI(note.GenerateID(), "A Protocol", note.TypeProtocol)
	concept := newTestNoteForCLI(note.GenerateID(), "A Concept", note.TypeConcept)
	writeNoteFile(t, nbDir, proto)
	writeNoteFile(t, nbDir, concept)

	out, err := execute("list", "--global")
	if err != nil {
		t.Fatalf("nn list --global: %v", err)
	}
	if strings.Contains(out, "A Concept") {
		t.Errorf("expected non-protocol note excluded from --global output:\n%s", out)
	}
}

// Assertion: --global with --type non-protocol returns an error.
func TestListGlobalWithNonProtocolTypeErrors(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("list", "--global", "--type", "concept")
	if err == nil {
		t.Fatal("nn list --global --type concept: want error, got nil")
	}
}

func assertCompactJSON(t *testing.T, out, label string) {
	t.Helper()
	if strings.Contains(out, "  \"") || strings.Contains(out, "{\n") {
		t.Errorf("%s: JSON output is not compact: found pretty-print markers", label)
	}
}

func TestListJSONCompact(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Alpha", note.TypeConcept))

	out, err := execute("list", "--json")
	if err != nil {
		t.Fatalf("nn list --json: %v", err)
	}
	assertCompactJSON(t, out, "nn list --json")

	out, err = execute("list", "--search", "Alpha", "--json")
	if err != nil {
		t.Fatalf("nn list --search --json: %v", err)
	}
	assertCompactJSON(t, out, "nn list --search --json")
}

func TestListSearchNeighborsJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	src := newTestNoteForCLI(note.GenerateID(), "Source Note", note.TypeConcept)
	src.Body = "uniqueterm alpha"
	dst := newTestNoteForCLI(note.GenerateID(), "Destination Note", note.TypeConcept)
	dst.Body = "target body"
	src.Links = []note.Link{
		{TargetID: dst.ID, Type: "supports", Annotation: "because it does"},
	}
	writeNoteFile(t, nbDir, dst)
	writeNoteFile(t, nbDir, src)

	out, err := execute("list", "--search", "uniqueterm", "--json")
	if err != nil {
		t.Fatalf("nn list --search uniqueterm --json: %v", err)
	}
	if !strings.Contains(out, `"neighbors"`) {
		t.Errorf("search JSON missing neighbors field; got: %s", out)
	}
	if !strings.Contains(out, `"direction"`) {
		t.Errorf("search JSON neighbors missing direction field; got: %s", out)
	}
	if !strings.Contains(out, "Destination Note") {
		t.Errorf("search JSON neighbors missing target title; got: %s", out)
	}
	if !strings.Contains(out, "supports") {
		t.Errorf("search JSON neighbors missing link type; got: %s", out)
	}
	if !strings.Contains(out, "because it does") {
		t.Errorf("search JSON neighbors missing annotation; got: %s", out)
	}
}

func TestListSearchNeighborsPlain(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	src := newTestNoteForCLI(note.GenerateID(), "Source Note Plain", note.TypeConcept)
	src.Body = "uniquetermplain"
	dst := newTestNoteForCLI(note.GenerateID(), "Destination Note Plain", note.TypeConcept)
	dst.Body = "target"
	src.Links = []note.Link{
		{TargetID: dst.ID, Type: "refines", Annotation: "sharpens the point"},
	}
	blinker := newTestNoteForCLI(note.GenerateID(), "Backlinker Note Plain", note.TypeConcept)
	blinker.Body = "another note"
	blinker.Links = []note.Link{
		{TargetID: src.ID, Type: "extends", Annotation: "builds on it"},
	}
	writeNoteFile(t, nbDir, dst)
	writeNoteFile(t, nbDir, blinker)
	writeNoteFile(t, nbDir, src)

	out, err := execute("list", "--search", "uniquetermplain")
	if err != nil {
		t.Fatalf("nn list --search uniquetermplain: %v", err)
	}
	if !strings.Contains(out, "→") {
		t.Errorf("plain search output missing outgoing neighbor arrow →; got: %s", out)
	}
	if !strings.Contains(out, "←") {
		t.Errorf("plain search output missing incoming neighbor arrow ←; got: %s", out)
	}
	if !strings.Contains(out, "Destination Note Plain") {
		t.Errorf("plain search output missing outgoing neighbor title; got: %s", out)
	}
	if !strings.Contains(out, "Backlinker Note Plain") {
		t.Errorf("plain search output missing incoming neighbor title; got: %s", out)
	}
}

func TestListSearchRelevanceOrder(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	now := time.Now().UTC().Truncate(time.Second)

	// low: newer created time (so created-desc sort puts it FIRST), but low BM25 relevance.
	low := newTestNoteForCLI(note.GenerateID(), "Unrelated Title Two", note.TypeConcept)
	low.Body = "zygote and many other words that dilute the term considerably here"
	low.Created = now
	low.Modified = now

	// high: older created time (so created-desc sort puts it SECOND), but high BM25 relevance.
	high := newTestNoteForCLI(note.GenerateID(), "Unrelated Title One", note.TypeConcept)
	high.Body = "zygote zygote zygote zygote zygote zygote zygote zygote zygote zygote"
	high.Created = now.Add(-time.Hour)
	high.Modified = now.Add(-time.Hour)

	writeNoteFile(t, nbDir, low)
	writeNoteFile(t, nbDir, high)

	out, err := execute("list", "--search", "zygote", "--json")
	if err != nil {
		t.Fatalf("nn list --search zygote --json: %v", err)
	}

	highIdx := strings.Index(out, high.ID)
	lowIdx := strings.Index(out, low.ID)
	if highIdx == -1 || lowIdx == -1 {
		t.Fatalf("expected both notes in output; high=%d low=%d\nout: %s", highIdx, lowIdx, out)
	}
	if highIdx > lowIdx {
		t.Errorf("search results not ordered by relevance: high-relevance note (id=%s) appeared after low-relevance note (id=%s)", high.ID, low.ID)
	}
}

func TestListSearchScoresDescending(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	now := time.Now().UTC().Truncate(time.Second)

	// Create notes with different backlink counts by linking — note with more
	// backlinks gets a centrality boost, so raw BM25 order ≠ normalized order.
	// We need the displayed score to match the sort order.
	a := newTestNoteForCLI(note.GenerateID(), "Alpha Zygote", note.TypeConcept)
	a.Body = "zygote zygote zygote zygote zygote zygote zygote"
	a.Created = now.Add(-2 * time.Hour)
	a.Modified = now.Add(-2 * time.Hour)

	b := newTestNoteForCLI(note.GenerateID(), "Beta Zygote", note.TypeConcept)
	b.Body = "zygote zygote zygote"
	b.Created = now.Add(-time.Hour)
	b.Modified = now.Add(-time.Hour)

	// c has lowest BM25 but many inbound links — centrality boost may push its
	// normalized score above b; sort must use normalized score to stay consistent
	// with what is displayed.
	c := newTestNoteForCLI(note.GenerateID(), "Gamma Zygote", note.TypeConcept)
	c.Body = "zygote"
	c.Created = now
	c.Modified = now

	// Give c many inbound links by creating linker notes that point to it.
	for range 10 {
		linker := newTestNoteForCLI(note.GenerateID(), "Linker", note.TypeConcept)
		linker.Links = []note.Link{{TargetID: c.ID, Type: "extends"}}
		writeNoteFile(t, nbDir, linker)
	}

	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)
	writeNoteFile(t, nbDir, c)

	out, err := execute("list", "--search", "zygote", "--json", "--fields", "id,score")
	if err != nil {
		t.Fatalf("nn list --search zygote --json: %v", err)
	}

	var results []struct {
		Score float64 `json:"score"`
	}
	mustJSON(t, out, &results)

	if len(results) < 2 {
		t.Fatalf("expected at least 2 results, got %d", len(results))
	}
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("scores not descending at index %d: score[%d]=%f > score[%d]=%f\nfull output: %s",
				i, i, results[i].Score, i-1, results[i-1].Score, out)
		}
	}
}

func TestListSearchDefaultLimit(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	now := time.Now().UTC().Truncate(time.Second)

	// Create 25 notes all matching "zygote"
	for i := range 25 {
		n := newTestNoteForCLI(note.GenerateID(), "Note", note.TypeConcept)
		n.Body = "zygote"
		n.Created = now.Add(time.Duration(-i) * time.Minute)
		n.Modified = n.Created
		writeNoteFile(t, nbDir, n)
	}

	out, err := execute("list", "--search", "zygote", "--json", "--fields", "id")
	if err != nil {
		t.Fatalf("nn list --search zygote --json: %v", err)
	}

	var results []struct {
		ID string `json:"id"`
	}
	mustJSON(t, out, &results)

	if len(results) > 20 {
		t.Errorf("default search limit: expected ≤20 results, got %d", len(results))
	}
}

func TestListSearchTruncationNotice(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	now := time.Now().UTC().Truncate(time.Second)

	for i := range 25 {
		n := newTestNoteForCLI(note.GenerateID(), "Note", note.TypeConcept)
		n.Body = "zygote"
		n.Created = now.Add(time.Duration(-i) * time.Minute)
		n.Modified = n.Created
		writeNoteFile(t, nbDir, n)
	}

	out, err := execute("list", "--search", "zygote")
	if err != nil {
		t.Fatalf("nn list --search zygote: %v", err)
	}

	if !strings.Contains(out, "more") || !strings.Contains(out, "--limit 0") {
		t.Errorf("expected truncation notice in plain-text output; got:\n%s", out)
	}
}

func TestListJSONExcerptTruncation(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	n := newTestNoteForCLI(note.GenerateID(), "Truncation Test", note.TypeConcept)
	// body longer than 200 chars so excerpt exceeds limit
	n.Body = "searchtruncword " + strings.Repeat("x", 250)
	writeNoteFile(t, nbDir, n)

	out, err := execute("list", "--search", "searchtruncword", "--json")
	if err != nil {
		t.Fatalf("nn list --search --json: %v", err)
	}

	var results []noteSearchJSON
	mustJSON(t, out, &results)
	if len(results) == 0 {
		t.Fatalf("expected at least one result")
	}
	if len(results[0].Excerpt) > 203 { // 200 chars + "..."
		t.Errorf("default excerpt not truncated: len=%d, want ≤203", len(results[0].Excerpt))
	}
}

func TestListJSONAnnotationTruncation(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	src := newTestNoteForCLI(note.GenerateID(), "Annot Source", note.TypeConcept)
	src.Body = "searchannotword body text"
	dst := newTestNoteForCLI(note.GenerateID(), "Annot Dest", note.TypeConcept)
	dst.Body = "dest body"
	longAnnotation := strings.Repeat("a", 120)
	src.Links = []note.Link{
		{TargetID: dst.ID, Type: "supports", Annotation: longAnnotation},
	}
	writeNoteFile(t, nbDir, dst)
	writeNoteFile(t, nbDir, src)

	out, err := execute("list", "--search", "searchannotword", "--json")
	if err != nil {
		t.Fatalf("nn list --search --json: %v", err)
	}

	var results []noteSearchJSON
	mustJSON(t, out, &results)
	if len(results) == 0 {
		t.Fatalf("expected at least one result")
	}
	found := false
	for _, nb := range results[0].Neighbors {
		if nb.ID == dst.ID {
			found = true
			if len(nb.Annotation) > 83 { // 80 chars + "..."
				t.Errorf("default annotation not truncated: len=%d, want ≤83", len(nb.Annotation))
			}
		}
	}
	if !found {
		t.Errorf("neighbor %s not found in results", dst.ID)
	}
}

func TestListJSONFullFlag(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	src := newTestNoteForCLI(note.GenerateID(), "Full Flag Source", note.TypeConcept)
	src.Body = "searchfullflag " + strings.Repeat("y", 250)
	dst := newTestNoteForCLI(note.GenerateID(), "Full Flag Dest", note.TypeConcept)
	dst.Body = "dest"
	longAnnotation := strings.Repeat("b", 120)
	src.Links = []note.Link{
		{TargetID: dst.ID, Type: "supports", Annotation: longAnnotation},
	}
	writeNoteFile(t, nbDir, dst)
	writeNoteFile(t, nbDir, src)

	out, err := execute("list", "--search", "searchfullflag", "--json", "--full")
	if err != nil {
		t.Fatalf("nn list --search --json --full: %v", err)
	}

	var results []noteSearchJSON
	mustJSON(t, out, &results)
	if len(results) == 0 {
		t.Fatalf("expected at least one result")
	}
	if len(results[0].Excerpt) <= 120 {
		t.Errorf("--full excerpt unexpectedly short: len=%d, want >120 (default excerpt window)", len(results[0].Excerpt))
	}
	for _, nb := range results[0].Neighbors {
		if nb.ID == dst.ID && len(nb.Annotation) <= 80 {
			t.Errorf("--full annotation unexpectedly short: len=%d", len(nb.Annotation))
		}
	}
}
