package cmd

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/jaresty/nn/internal/note"
)

const graphBodiesTestTransportLimit = 50_000

type graphBodiesTestSegment struct {
	ID       string `json:"id"`
	Segment  int    `json:"segment"`
	Segments int    `json:"segments"`
	Body     string `json:"body"`
}

type graphBodiesTestPage struct {
	Snapshot string                   `json:"snapshot"`
	Page     int                      `json:"page"`
	Pages    int                      `json:"pages"`
	NextPage int                      `json:"next_page"`
	Segments []graphBodiesTestSegment `json:"segments"`
}

func fetchGraphBodyPages(t *testing.T, execute func(...string) (string, error), traversal ...string) ([]string, []graphBodiesTestPage) {
	t.Helper()
	firstArgs := append([]string{"graph", "bodies"}, traversal...)
	firstArgs = append(firstArgs, "--page", "1")
	firstOut, err := execute(firstArgs...)
	if err != nil {
		t.Fatalf("graph bodies page 1: %v", err)
	}
	var first graphBodiesTestPage
	if err := json.Unmarshal([]byte(firstOut), &first); err != nil {
		t.Fatalf("graph bodies page 1 JSON: %v\n%s", err, firstOut)
	}
	if first.Page != 1 || first.Pages < 1 {
		t.Fatalf("graph bodies first envelope = page %d of %d", first.Page, first.Pages)
	}
	if len(first.Snapshot) != 64 {
		t.Fatalf("snapshot length = %d, want 64: %q", len(first.Snapshot), first.Snapshot)
	}
	if _, err := hex.DecodeString(first.Snapshot); err != nil {
		t.Fatalf("snapshot is not lowercase SHA-256 hex: %q (%v)", first.Snapshot, err)
	}
	if first.Snapshot != strings.ToLower(first.Snapshot) {
		t.Fatalf("snapshot is not lowercase: %q", first.Snapshot)
	}

	outputs := []string{firstOut}
	pages := []graphBodiesTestPage{first}
	for page := 2; page <= first.Pages; page++ {
		args := append([]string{"graph", "bodies"}, traversal...)
		args = append(args, "--page", fmt.Sprint(page), "--snapshot", first.Snapshot)
		out, err := execute(args...)
		if err != nil {
			t.Fatalf("graph bodies page %d: %v", page, err)
		}
		var got graphBodiesTestPage
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatalf("graph bodies page %d JSON: %v\n%s", page, err, out)
		}
		if got.Snapshot != first.Snapshot || got.Page != page || got.Pages != first.Pages {
			t.Fatalf("page %d envelope = snapshot %q page %d of %d; want snapshot %q page %d of %d", page, got.Snapshot, got.Page, got.Pages, first.Snapshot, page, first.Pages)
		}
		outputs = append(outputs, out)
		pages = append(pages, got)
	}
	for i, page := range pages {
		wantNext := 0
		if i+1 < len(pages) {
			wantNext = i + 2
		}
		if page.NextPage != wantNext {
			t.Fatalf("page %d next_page = %d, want %d", page.Page, page.NextPage, wantNext)
		}
	}
	return outputs, pages
}

func storedGraphBody(t *testing.T, n *note.Note) string {
	t.Helper()
	data, err := n.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := note.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	return parsed.Body
}

func reconstructGraphBodies(t *testing.T, pages []graphBodiesTestPage) map[string]string {
	t.Helper()
	byID := make(map[string][]graphBodiesTestSegment)
	for _, page := range pages {
		for _, segment := range page.Segments {
			if !utf8.ValidString(segment.Body) {
				t.Fatalf("note %s segment %d is not valid UTF-8", segment.ID, segment.Segment)
			}
			byID[segment.ID] = append(byID[segment.ID], segment)
		}
	}
	bodies := make(map[string]string, len(byID))
	for id, segments := range byID {
		for i, segment := range segments {
			if segment.Segment != i+1 {
				t.Fatalf("note %s segment ordinal at position %d = %d", id, i, segment.Segment)
			}
			if segment.Segments != len(segments) {
				t.Fatalf("note %s segment %d reports %d total, received %d", id, segment.Segment, segment.Segments, len(segments))
			}
			bodies[id] += segment.Body
		}
	}
	return bodies
}

// retained property [28]: graph bodies selects exactly the same IDs as graph
// show for focus/depth/direction/link/status/representation constrained traversal.
func TestGraphBodiesFilterParityWithGraphShow(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root := graphShowFilterNote("body root", note.StatusDraft, "")
	root.ID = "20990101000000-9000"
	reviewedTax := graphShowFilterNote("reviewed taxonomy", note.StatusReviewed, "taxonomy")
	reviewedTax.ID = "20990101000000-1000"
	deep := graphShowFilterNote("deep permanent taxonomy", note.StatusPermanent, "taxonomy")
	deep.ID = "20990101000000-2000"
	wrongStatus := graphShowFilterNote("draft taxonomy", note.StatusDraft, "taxonomy")
	wrongStatus.ID = "20990101000000-3000"
	wrongRep := graphShowFilterNote("reviewed ontology", note.StatusReviewed, "ontology")
	wrongRep.ID = "20990101000000-4000"
	incoming := graphShowFilterNote("incoming taxonomy", note.StatusReviewed, "taxonomy")
	incoming.ID = "20990101000000-5000"

	root.Links = []note.Link{
		{TargetID: reviewedTax.ID, Type: "supports"},
		{TargetID: wrongStatus.ID, Type: "supports"},
		{TargetID: wrongRep.ID, Type: "supports"},
	}
	reviewedTax.Links = []note.Link{{TargetID: deep.ID, Type: "governs"}}
	incoming.Links = []note.Link{{TargetID: root.ID, Type: "supports"}}
	for _, n := range []*note.Note{root, reviewedTax, deep, wrongStatus, wrongRep, incoming} {
		n.Body = "body for " + n.ID
		writeNoteFile(t, nbDir, n)
	}

	cases := [][]string{
		{"--focus", root.ID, "--depth", "2", "--direction", "both"},
		{"--focus", root.ID, "--depth", "2", "--direction", "outgoing", "--links", "supports,governs", "--status", "reviewed,permanent", "--representation", "taxonomy"},
	}
	for _, traversal := range cases {
		showArgs := append([]string{"graph", "show", "--format", "json"}, traversal...)
		showOut, err := execute(showArgs...)
		if err != nil {
			t.Fatalf("graph show parity source: %v", err)
		}
		var show graphShowFilterResult
		if err := json.Unmarshal([]byte(showOut), &show); err != nil {
			t.Fatalf("graph show parity JSON: %v", err)
		}
		var want []string
		for _, n := range show.Nodes {
			want = append(want, n.ID)
		}
		sort.Strings(want)

		_, pages := fetchGraphBodyPages(t, execute, traversal...)
		bodies := reconstructGraphBodies(t, pages)
		got := make([]string, 0, len(bodies))
		for id := range bodies {
			got = append(got, id)
		}
		sort.Strings(got)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("graph bodies IDs = %v, graph show IDs = %v for %v", got, want, traversal)
		}
	}
}

// retained properties [29]-[30]: JSON overhead is included in the page cap;
// Unicode and huge/empty bodies reconstruct byte-for-byte from ordered segments.
func TestGraphBodiesLosslessUnicodeHugeAndEmptyWithinTransportCap(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	huge := newTestNoteForCLI("20990101000000-1000", "Huge Unicode", note.TypeConcept)
	empty := newTestNoteForCLI("20990101000000-2000", "Empty", note.TypeObservation)
	small := newTestNoteForCLI("20990101000000-3000", "Small Unicode", note.TypeArgument)
	huge.Body = strings.Repeat("🙂<&> 雪 e\u0301\nJSON \\\" boundary\t", 9_000) + "終"
	empty.Body = ""
	small.Body = "first\r\nsecond\nمرحبا\nनमस्ते\nemoji: 🧭"
	for _, n := range []*note.Note{small, empty, huge} {
		writeNoteFile(t, nbDir, n)
	}

	outputs, pages := fetchGraphBodyPages(t, execute)
	if len(pages) < 2 {
		t.Fatalf("huge body fit in %d page; want pagination", len(pages))
	}
	for i, out := range outputs {
		if len([]byte(out)) >= graphBodiesTestTransportLimit {
			t.Fatalf("page %d encoded size = %d bytes, want safely below %d", i+1, len([]byte(out)), graphBodiesTestTransportLimit)
		}
	}
	var orderedKeys []string
	for _, page := range pages {
		for _, segment := range page.Segments {
			orderedKeys = append(orderedKeys, fmt.Sprintf("%s/%09d", segment.ID, segment.Segment))
		}
	}
	if !sort.StringsAreSorted(orderedKeys) {
		t.Fatalf("body records are not ordered by note ID then segment: %v", orderedKeys)
	}
	bodies := reconstructGraphBodies(t, pages)
	want := map[string]string{huge.ID: storedGraphBody(t, huge), empty.ID: storedGraphBody(t, empty), small.ID: storedGraphBody(t, small)}
	for id, expected := range want {
		if !bytes.Equal([]byte(bodies[id]), []byte(expected)) {
			t.Fatalf("note %s reconstructed bytes differ: got %d bytes, want %d", id, len([]byte(bodies[id])), len([]byte(expected)))
		}
	}
	if segments := countGraphBodySegments(pages, huge.ID); segments < 2 {
		t.Fatalf("huge note has %d segment, want multiple", segments)
	}
	if segments := countGraphBodySegments(pages, empty.ID); segments != 1 {
		t.Fatalf("empty note has %d segments, want exactly one", segments)
	}
	for _, page := range pages {
		for _, segment := range page.Segments {
			if segment.ID == empty.ID && (segment.Body != "" || segment.Segment != 1 || segment.Segments != 1) {
				t.Fatalf("empty body representation = %#v", segment)
			}
		}
	}
}

func countGraphBodySegments(pages []graphBodiesTestPage, id string) int {
	count := 0
	for _, page := range pages {
		for _, segment := range page.Segments {
			if segment.ID == id {
				count++
			}
		}
	}
	return count
}

// retained property [31]: pagination is deterministic and every later page is
// bound to the repeated snapshot; notebook or traversal changes reject the token.
func TestGraphBodiesDeterministicSnapshotAndStaleRejection(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root := newTestNoteForCLI("20990101000000-1000", "Snapshot root", note.TypeModel)
	child := newTestNoteForCLI("20990101000000-2000", "Snapshot child", note.TypeConcept)
	outsider := newTestNoteForCLI("20990101000000-3000", "Snapshot outsider", note.TypeObservation)
	root.Body = strings.Repeat("root🙂<&>\n", 12_000)
	child.Body = "child"
	outsider.Body = "outside selected traversal"
	root.Links = []note.Link{{TargetID: child.ID, Type: "supports"}}
	for _, n := range []*note.Note{root, child, outsider} {
		writeNoteFile(t, nbDir, n)
	}
	traversal := []string{"--focus", root.ID, "--depth", "2", "--direction", "outgoing", "--links", "supports,governs"}
	firstOutputs, firstPages := fetchGraphBodyPages(t, execute, traversal...)
	secondOutputs, secondPages := fetchGraphBodyPages(t, execute, traversal...)
	if !reflect.DeepEqual(firstOutputs, secondOutputs) || !reflect.DeepEqual(firstPages, secondPages) {
		t.Fatal("unchanged graph bodies pagination is not byte-for-byte deterministic")
	}
	if len(firstPages) < 2 {
		t.Fatal("snapshot fixture did not produce a later page")
	}
	snapshot := firstPages[0].Snapshot

	withoutSnapshot := append([]string{"graph", "bodies"}, traversal...)
	withoutSnapshot = append(withoutSnapshot, "--page", "2")
	if _, err := execute(withoutSnapshot...); err == nil || !strings.Contains(err.Error(), "--snapshot is required") {
		t.Fatalf("later page without snapshot error = %v", err)
	}

	// Set-valued CSV filters are normalized, so equivalent ordering identifies
	// the same traversal request and may continue the same snapshot.
	equivalentTraversal := []string{"graph", "bodies", "--focus", root.ID, "--depth", "2", "--direction", "outgoing", "--links", "governs,supports", "--page", "2", "--snapshot", snapshot}
	if _, err := execute(equivalentTraversal...); err != nil {
		t.Fatalf("equivalent normalized traversal rejected snapshot: %v", err)
	}

	mismatchedTraversal := []string{"graph", "bodies", "--focus", root.ID, "--depth", "1", "--direction", "outgoing", "--links", "supports,governs", "--page", "2", "--snapshot", snapshot}
	if _, err := execute(mismatchedTraversal...); err == nil || !strings.Contains(err.Error(), "stale or mismatched") {
		t.Fatalf("mismatched traversal snapshot error = %v", err)
	}

	// Mutating a note outside the filtered traversal still invalidates the
	// notebook snapshot rather than mixing pages from two notebook states.
	outsider.Body = "changed outside selected traversal"
	outsider.Modified = outsider.Modified.Add(1)
	writeNoteFile(t, nbDir, outsider)
	staleArgs := append([]string{"graph", "bodies"}, traversal...)
	staleArgs = append(staleArgs, "--page", "2", "--snapshot", snapshot)
	if _, err := execute(staleArgs...); err == nil || !strings.Contains(err.Error(), "stale or mismatched") {
		t.Fatalf("stale notebook snapshot error = %v", err)
	}
}

// retained property [32]: graph bodies preserves the stored-body metadata
// boundary; transport records expose no frontmatter, links, topology, or hints.
func TestGraphBodiesPreservesMetadataBoundary(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI("20990101000000-1000", "Metadata title must stay topology-side", note.TypeProtocol)
	n.Status = note.StatusPermanent
	n.Tags = []string{"secret-tag"}
	n.Representation = "ontology"
	n.AppliesWhen = "metadata condition"
	n.Body = "stored body only"
	n.Links = []note.Link{{TargetID: "20990101000000-9999", Type: "governs", Annotation: "metadata edge"}}
	writeNoteFile(t, nbDir, n)
	outputs, _ := fetchGraphBodyPages(t, execute)

	for _, out := range outputs {
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal([]byte(out), &envelope); err != nil {
			t.Fatal(err)
		}
		wantEnvelope := []string{"next_page", "page", "pages", "segments", "snapshot"}
		var gotEnvelope []string
		for key := range envelope {
			gotEnvelope = append(gotEnvelope, key)
		}
		sort.Strings(gotEnvelope)
		if !reflect.DeepEqual(gotEnvelope, wantEnvelope) {
			t.Fatalf("envelope fields = %v, want %v", gotEnvelope, wantEnvelope)
		}
		var records []map[string]json.RawMessage
		if err := json.Unmarshal(envelope["segments"], &records); err != nil {
			t.Fatal(err)
		}
		for _, record := range records {
			wantRecord := []string{"body", "id", "segment", "segments"}
			var gotRecord []string
			for key := range record {
				gotRecord = append(gotRecord, key)
			}
			sort.Strings(gotRecord)
			if !reflect.DeepEqual(gotRecord, wantRecord) {
				t.Fatalf("body record fields = %v, want %v", gotRecord, wantRecord)
			}
		}
	}
}

func TestGraphBodiesValidationAndFullGraphDefaults(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI("20990101000000-1000", "Only note", note.TypeConcept)
	n.Body = "body"
	writeNoteFile(t, nbDir, n)
	_, pages := fetchGraphBodyPages(t, execute)
	if got := reconstructGraphBodies(t, pages); got[n.ID] != storedGraphBody(t, n) {
		t.Fatalf("full graph bodies = %v", got)
	}

	for _, args := range [][]string{
		{"graph", "bodies", "--depth", "1"},
		{"graph", "bodies", "--direction", "both"},
		{"graph", "bodies", "--links", "supports"},
		{"graph", "bodies", "--status", "reviewed"},
		{"graph", "bodies", "--representation", "taxonomy"},
	} {
		if _, err := execute(args...); err == nil || !strings.Contains(err.Error(), "requires --focus") {
			t.Fatalf("no-focus traversal args %v error = %v", args, err)
		}
	}
	for _, args := range [][]string{
		{"graph", "bodies", "--page", "0"},
		{"graph", "bodies", "--page", "2", "--snapshot", strings.Repeat("0", 64)},
		{"graph", "bodies", "--focus", "missing"},
	} {
		if _, err := execute(args...); err == nil {
			t.Fatalf("invalid graph bodies args accepted: %v", args)
		}
	}
}

// retained property [33]: Cobra deprecates graph show --bodies in help and on
// stderr while its compatibility stdout remains the existing complete output.
func TestGraphShowBodiesDeprecationPreservesCompatibilityOutput(t *testing.T) {
	nbDir, cfgFile := setupNotebookWithCfg(t)
	n := newTestNoteForCLI("20990101000000-1000", "Compatibility", note.TypeConcept)
	n.Body = "complete legacy body\nwith a second line"
	writeNoteFile(t, nbDir, n)

	stdout, stderr, err := executeWithStderr(t, cfgFile, "graph", "show", "--bodies", "--format", "json")
	if err != nil {
		t.Fatalf("deprecated graph show --bodies: %v", err)
	}
	if !strings.Contains(stderr, "deprecated") || !strings.Contains(stderr, "graph bodies") {
		t.Fatalf("deprecation stderr = %q", stderr)
	}
	var got struct {
		Nodes []struct {
			ID   string `json:"id"`
			Body string `json:"body"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("legacy stdout is not clean JSON: %v\n%s", err, stdout)
	}
	if len(got.Nodes) != 1 || got.Nodes[0].ID != n.ID || got.Nodes[0].Body != storedGraphBody(t, n) {
		t.Fatalf("legacy body output changed: %#v", got.Nodes)
	}

	help, _, err := executeWithStderr(t, cfgFile, "graph", "show", "--help")
	if err != nil {
		t.Fatalf("graph show help: %v", err)
	}
	if !strings.Contains(strings.ToLower(help), "deprecated") || !strings.Contains(help, "nn graph bodies") {
		t.Fatalf("graph show help lacks body deprecation:\n%s", help)
	}
}

func TestGraphBodiesRejectsInvalidUTF8RatherThanReplacingBytes(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	n := newTestNoteForCLI("20990101000000-1000", "Invalid UTF-8", note.TypeConcept)
	writeNoteFile(t, nbDir, n)
	path := filepath.Join(nbDir, n.Filename())
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.Replace(data, []byte("Test body."), []byte{'o', 'k', 0xff, 'x'}, 1)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := execute("graph", "bodies"); err == nil || !strings.Contains(err.Error(), "valid UTF-8") {
		t.Fatalf("invalid UTF-8 error = %v", err)
	}
}
