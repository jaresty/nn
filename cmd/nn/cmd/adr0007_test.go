package cmd

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

// ── Feature 1: nn list --long ─────────────────────────────────────────────────

func TestListLong(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	short := newTestNoteForCLI(note.GenerateID(), "Short Note", note.TypeConcept)
	short.Body = "Small body."

	long := newTestNoteForCLI(note.GenerateID(), "Long Note", note.TypeConcept)
	long.Body = strings.Repeat("x", atomicityThreshold+1)

	writeNoteFile(t, nbDir, short)
	writeNoteFile(t, nbDir, long)

	out, err := execute("list", "--long")
	if err != nil {
		t.Fatalf("nn list --long: %v", err)
	}
	if !strings.Contains(out, long.ID) {
		t.Errorf("list --long missing long note %q:\n%s", long.ID, out)
	}
	if strings.Contains(out, short.ID) {
		t.Errorf("list --long should exclude short note %q:\n%s", short.ID, out)
	}
}

func TestListLongJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	long := newTestNoteForCLI(note.GenerateID(), "Long Note JSON", note.TypeConcept)
	long.Body = strings.Repeat("y", atomicityThreshold+1)
	writeNoteFile(t, nbDir, long)

	out, err := execute("list", "--long", "--json")
	if err != nil {
		t.Fatalf("nn list --long --json: %v", err)
	}
	var results []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(out), &results); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(results) != 1 || results[0].ID != long.ID {
		t.Errorf("expected [%s], got %v", long.ID, results)
	}
}

// ── Feature 2: nn status long notes ──────────────────────────────────────────

func TestStatusLongNotes(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	short := newTestNoteForCLI(note.GenerateID(), "Short", note.TypeConcept)
	short.Body = "small"

	lng := newTestNoteForCLI(note.GenerateID(), "Dense Concept Note", note.TypeConcept)
	lng.Body = strings.Repeat("z", atomicityThreshold+1)

	writeNoteFile(t, nbDir, short)
	writeNoteFile(t, nbDir, lng)

	out, err := execute("status")
	if err != nil {
		t.Fatalf("nn status: %v", err)
	}
	if !strings.Contains(out, "long notes") {
		t.Errorf("status missing 'long notes' section:\n%s", out)
	}
	if !strings.Contains(out, lng.ID) {
		t.Errorf("status missing long note ID %q:\n%s", lng.ID, out)
	}
}

func TestStatusLongNotesAbsentWhenNone(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	short := newTestNoteForCLI(note.GenerateID(), "Short", note.TypeConcept)
	short.Body = "tiny"
	writeNoteFile(t, nbDir, short)

	out, err := execute("status")
	if err != nil {
		t.Fatalf("nn status: %v", err)
	}
	if strings.Contains(out, "long notes") {
		t.Errorf("status should omit 'long notes' when none exist:\n%s", out)
	}
}

func TestStatusLongNotesJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	lng := newTestNoteForCLI(note.GenerateID(), "Big Note", note.TypeConcept)
	lng.Body = strings.Repeat("w", atomicityThreshold+1)
	writeNoteFile(t, nbDir, lng)

	out, err := execute("status", "--json")
	if err != nil {
		t.Fatalf("nn status --json: %v", err)
	}
	var result struct {
		LongNotes []struct {
			ID      string `json:"id"`
			Title   string `json:"title"`
			BodyLen int    `json:"body_len"`
		} `json:"long_notes"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(result.LongNotes) != 1 {
		t.Errorf("expected 1 long note, got %d", len(result.LongNotes))
	} else {
		if result.LongNotes[0].ID != lng.ID {
			t.Errorf("long note ID: got %q want %q", result.LongNotes[0].ID, lng.ID)
		}
		if result.LongNotes[0].BodyLen <= atomicityThreshold {
			t.Errorf("body_len %d should exceed %d", result.LongNotes[0].BodyLen, atomicityThreshold)
		}
	}
}

// ── Feature 3: nn status hub notes ───────────────────────────────────────────

func TestStatusHubNotes(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// Create 10+ notes so hub section appears.
	notes := make([]*note.Note, 12)
	for i := range notes {
		n := newTestNoteForCLI(note.GenerateID(), "Note", note.TypeConcept)
		notes[i] = n
		writeNoteFile(t, nbDir, n)
	}

	// Hub: notes[0] is linked from many others.
	hub := notes[0]
	for i := 1; i < 8; i++ {
		notes[i].Links = []note.Link{{TargetID: hub.ID, Annotation: "links to hub", Type: "extends"}}
		writeNoteFile(t, nbDir, notes[i])
	}

	out, err := execute("status")
	if err != nil {
		t.Fatalf("nn status: %v", err)
	}
	if !strings.Contains(out, "hub notes") {
		t.Errorf("status missing 'hub notes' section:\n%s", out)
	}
	if !strings.Contains(out, hub.ID) {
		t.Errorf("status hub notes missing hub ID %q:\n%s", hub.ID, out)
	}
}

func TestStatusHubNotesAbsentWhenSparse(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	// Only 5 notes — below threshold.
	for i := 0; i < 5; i++ {
		writeNoteFile(t, nbDir, newTestNoteForCLI(note.GenerateID(), "Note", note.TypeConcept))
	}

	out, err := execute("status")
	if err != nil {
		t.Fatalf("nn status: %v", err)
	}
	if strings.Contains(out, "hub notes") {
		t.Errorf("status should omit 'hub notes' when fewer than 10 notes:\n%s", out)
	}
}

func TestStatusHubNotesJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	notes := make([]*note.Note, 10)
	for i := range notes {
		n := newTestNoteForCLI(note.GenerateID(), "Note", note.TypeConcept)
		notes[i] = n
		writeNoteFile(t, nbDir, n)
	}
	hub := notes[0]
	for i := 1; i < 6; i++ {
		notes[i].Links = []note.Link{{TargetID: hub.ID, Annotation: "links hub", Type: "extends"}}
		writeNoteFile(t, nbDir, notes[i])
	}

	out, err := execute("status", "--json")
	if err != nil {
		t.Fatalf("nn status --json: %v", err)
	}
	var result struct {
		HubNotes []struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			Degree int    `json:"degree"`
		} `json:"hub_notes"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(result.HubNotes) == 0 {
		t.Errorf("expected hub notes in JSON, got none")
	}
	found := false
	for _, h := range result.HubNotes {
		if h.ID == hub.ID {
			found = true
			if h.Degree < 5 {
				t.Errorf("hub degree: got %d, want >= 5", h.Degree)
			}
		}
	}
	if !found {
		t.Errorf("hub note %q not found in hub_notes", hub.ID)
	}
}

// ── Feature 4: nn path ────────────────────────────────────────────────────────

func TestPathDirectLink(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	a := newTestNoteForCLI(note.GenerateID(), "Note A", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "Note B", note.TypeConcept)
	a.Links = []note.Link{{TargetID: b.ID, Annotation: "links to b", Type: "extends"}}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	out, err := execute("path", a.ID, b.ID)
	if err != nil {
		t.Fatalf("nn path: %v", err)
	}
	if !strings.Contains(out, a.ID) || !strings.Contains(out, b.ID) {
		t.Errorf("path output missing IDs:\n%s", out)
	}
}

func TestPathTwoHops(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	a := newTestNoteForCLI(note.GenerateID(), "A", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "B", note.TypeConcept)
	c := newTestNoteForCLI(note.GenerateID(), "C", note.TypeConcept)
	a.Links = []note.Link{{TargetID: b.ID, Annotation: "a->b", Type: "extends"}}
	b.Links = []note.Link{{TargetID: c.ID, Annotation: "b->c", Type: "extends"}}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)
	writeNoteFile(t, nbDir, c)

	out, err := execute("path", a.ID, c.ID)
	if err != nil {
		t.Fatalf("nn path: %v", err)
	}
	if !strings.Contains(out, b.ID) {
		t.Errorf("path should include intermediate B %q:\n%s", b.ID, out)
	}
}

func TestPathNoPath(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	a := newTestNoteForCLI(note.GenerateID(), "A", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "B", note.TypeConcept)
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	_, err := execute("path", a.ID, b.ID)
	if err == nil {
		t.Error("nn path with no path should return error")
	}
}

func TestPathJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	a := newTestNoteForCLI(note.GenerateID(), "A", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "B", note.TypeConcept)
	a.Links = []note.Link{{TargetID: b.ID, Annotation: "a->b", Type: "extends"}}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	out, err := execute("path", a.ID, b.ID, "--json")
	if err != nil {
		t.Fatalf("nn path --json: %v", err)
	}
	var steps []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	if err := json.Unmarshal([]byte(out), &steps); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(steps))
	}
}

// ── Feature 5: nn clusters ────────────────────────────────────────────────────

func TestClustersBasic(t *testing.T) {
	nbDir, execute := setupNotebook(t)

	// Create two clusters of linked notes.
	clusterA := make([]*note.Note, 3)
	for i := range clusterA {
		clusterA[i] = newTestNoteForCLI(note.GenerateID(), "A Note", note.TypeConcept)
		writeNoteFile(t, nbDir, clusterA[i])
	}
	clusterA[0].Links = []note.Link{{TargetID: clusterA[1].ID, Annotation: "link", Type: "extends"}}
	clusterA[1].Links = []note.Link{{TargetID: clusterA[2].ID, Annotation: "link", Type: "extends"}}
	writeNoteFile(t, nbDir, clusterA[0])
	writeNoteFile(t, nbDir, clusterA[1])

	clusterB := make([]*note.Note, 3)
	for i := range clusterB {
		clusterB[i] = newTestNoteForCLI(note.GenerateID(), "B Note", note.TypeConcept)
		writeNoteFile(t, nbDir, clusterB[i])
	}
	clusterB[0].Links = []note.Link{{TargetID: clusterB[1].ID, Annotation: "link", Type: "extends"}}
	clusterB[1].Links = []note.Link{{TargetID: clusterB[2].ID, Annotation: "link", Type: "extends"}}
	writeNoteFile(t, nbDir, clusterB[0])
	writeNoteFile(t, nbDir, clusterB[1])

	out, err := execute("clusters")
	if err != nil {
		t.Fatalf("nn clusters: %v", err)
	}
	if !strings.Contains(out, "cluster") {
		t.Errorf("clusters output missing 'cluster':\n%s", out)
	}
	// Text output must show note IDs alongside titles.
	if !strings.Contains(out, clusterA[0].ID) {
		t.Errorf("clusters text output missing note ID %q:\n%s", clusterA[0].ID, out)
	}
}

func TestClustersJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	a := newTestNoteForCLI(note.GenerateID(), "A", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "B", note.TypeConcept)
	a.Links = []note.Link{{TargetID: b.ID, Annotation: "link", Type: "extends"}}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	out, err := execute("clusters", "--json")
	if err != nil {
		t.Fatalf("nn clusters --json: %v", err)
	}
	var clusters []struct {
		Notes []struct {
			ID    string `json:"id"`
			Title string `json:"title"`
		} `json:"notes"`
	}
	if err := json.Unmarshal([]byte(out), &clusters); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if len(clusters) == 0 {
		t.Error("expected at least one cluster")
	}
}

func TestClustersMinFilter(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	// Two connected notes (cluster of 2) and one singleton.
	a := newTestNoteForCLI(note.GenerateID(), "A", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "B", note.TypeConcept)
	singleton := newTestNoteForCLI(note.GenerateID(), "Singleton", note.TypeConcept)
	a.Links = []note.Link{{TargetID: b.ID, Annotation: "link", Type: "extends"}}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)
	writeNoteFile(t, nbDir, singleton)

	// --min 2 should show the pair; --singletons would be needed for singleton.
	out, err := execute("clusters", "--min", "2")
	if err != nil {
		t.Fatalf("nn clusters --min 2: %v", err)
	}
	// The singleton should not appear.
	if strings.Contains(out, singleton.ID) {
		t.Errorf("clusters --min 2 should exclude singleton %q:\n%s", singleton.ID, out)
	}
}

// ── Feature 6: nn new --from-stdin ───────────────────────────────────────────

func TestNewFromStdin(t *testing.T) {
	_, cfgFile := setupNotebookWithCfg(t)

	var buf strings.Builder
	buf.WriteString("# Stdin Content\n\nThis came from stdin.")

	root := NewRootCmdForTest(cfgFile)
	root.SetArgs([]string{"new", "--title", "Stdin Note", "--type", "concept", "--from-stdin"})
	root.SetIn(strings.NewReader(buf.String()))
	var stdout strings.Builder
	root.SetOut(&stdout)

	if err := root.Execute(); err != nil {
		t.Fatalf("nn new --from-stdin: %v", err)
	}
	if !strings.Contains(stdout.String(), "created") {
		t.Errorf("expected 'created' in output: %q", stdout.String())
	}
}

// ── Feature 7: Link status (draft/reviewed) ───────────────────────────────────

func TestLinkStatusDraftDefault(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	a := newTestNoteForCLI(note.GenerateID(), "A", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "B", note.TypeConcept)
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	// nn link defaults to draft — need to use the backend through CLI.
	// Write notes with links directly for this test.
	a.Links = []note.Link{{TargetID: b.ID, Annotation: "test link", Type: "extends", Status: "draft"}}
	writeNoteFile(t, nbDir, a)

	out, err := execute("links", a.ID)
	if err != nil {
		t.Fatalf("nn links: %v", err)
	}
	if !strings.Contains(out, "draft") {
		t.Errorf("links output should show 'draft' status:\n%s", out)
	}
}

func TestLinkStatusFilterDraft(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	a := newTestNoteForCLI(note.GenerateID(), "A", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "B", note.TypeConcept)
	c := newTestNoteForCLI(note.GenerateID(), "C", note.TypeConcept)

	a.Links = []note.Link{
		{TargetID: b.ID, Annotation: "draft link", Type: "extends", Status: "draft"},
		{TargetID: c.ID, Annotation: "reviewed link", Type: "extends", Status: "reviewed"},
	}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)
	writeNoteFile(t, nbDir, c)

	out, err := execute("links", a.ID, "--status", "draft")
	if err != nil {
		t.Fatalf("nn links --status draft: %v", err)
	}
	if !strings.Contains(out, b.ID) {
		t.Errorf("expected draft target %q in output:\n%s", b.ID, out)
	}
	if strings.Contains(out, c.ID) {
		t.Errorf("reviewed target %q should be excluded:\n%s", c.ID, out)
	}
}

func TestStatusDraftLinkCount(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	a := newTestNoteForCLI(note.GenerateID(), "A", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "B", note.TypeConcept)
	a.Links = []note.Link{{TargetID: b.ID, Annotation: "draft link", Type: "extends", Status: "draft"}}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	out, err := execute("status")
	if err != nil {
		t.Fatalf("nn status: %v", err)
	}
	if !strings.Contains(out, "draft link") {
		t.Errorf("status should report draft links count:\n%s", out)
	}
}

func TestStatusDraftLinkCountJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	a := newTestNoteForCLI(note.GenerateID(), "A", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "B", note.TypeConcept)
	a.Links = []note.Link{{TargetID: b.ID, Annotation: "draft link", Type: "extends", Status: "draft"}}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	out, err := execute("status", "--json")
	if err != nil {
		t.Fatalf("nn status --json: %v", err)
	}
	var result struct {
		DraftLinks int `json:"draft_links"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if result.DraftLinks != 1 {
		t.Errorf("draft_links: got %d, want 1", result.DraftLinks)
	}
}

func TestLinkStatusRoundtrip(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	a := newTestNoteForCLI(note.GenerateID(), "A", note.TypeConcept)
	b := newTestNoteForCLI(note.GenerateID(), "B", note.TypeConcept)
	// Write a link with reviewed status (no-status legacy = reviewed).
	a.Links = []note.Link{{TargetID: b.ID, Annotation: "old link", Type: "extends"}}
	writeNoteFile(t, nbDir, a)
	writeNoteFile(t, nbDir, b)

	out, err := execute("links", a.ID, "--status", "reviewed")
	if err != nil {
		t.Fatalf("nn links --status reviewed: %v", err)
	}
	// Legacy link (no status) should be treated as reviewed.
	if !strings.Contains(out, b.ID) {
		t.Errorf("legacy (no-status) link should appear as reviewed:\n%s", out)
	}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

// newTestNoteWithBody creates a test note with a specific body.
func newTestNoteWithBody(id, title string, typ note.Type, body string) *note.Note {
	n := newTestNoteForCLI(id, title, typ)
	n.Body = body
	n.Modified = time.Now().UTC().Truncate(time.Second)
	return n
}
