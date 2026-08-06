package cmd

import (
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestEscapeMermaidLabelUsesMermaidNumericEntities(t *testing.T) {
	input := "#\"&<>\n\r\\|"
	want := "#35;#34;#38;#60;#62;#10;#13;#92;#124;"
	if got := escapeMermaidLabel(input); got != want {
		t.Fatalf("Mermaid numeric escaping = %q, want %q", got, want)
	}
}

func TestGraphShowDepthRequiresFocusAndFullGraphIsDocumented(t *testing.T) {
	_, execute := setupNotebook(t)
	_, err := execute("graph", "show", "--depth", "3")
	if err == nil {
		t.Fatal("no-focus --depth accepted")
	}
	if !strings.Contains(err.Error(), "--depth requires --focus") {
		t.Fatalf("no-focus --depth error = %q, want requires --focus", err)
	}
	virtual, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("virtual CLI reference: %v", err)
	}
	guide, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-guide: %v", err)
	}
	for name, content := range map[string]string{"virtual CLI reference": virtual, "nn-guide": string(guide)} {
		for _, required := range []string{"[--focus <id>]", "Without --focus, graph show renders the full graph"} {
			if !strings.Contains(content, required) {
				t.Fatalf("%s missing no-focus contract %q", name, required)
			}
		}
	}
}

func TestGraphShowMermaidNoFocusProvenanceIsFullScope(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("graph", "show", "--format", "mermaid")
	if err != nil {
		t.Fatalf("no-focus Mermaid provenance: %v", err)
	}
	firstLine := strings.SplitN(out, "\n", 2)[0]
	if firstLine != "%% nn graph show scope=full" {
		t.Fatalf("no-focus Mermaid provenance = %q, want full scope", firstLine)
	}
}

func TestGraphShowMermaidFocusedBrokenEdgeMatchesFilteredJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	source := graphShowFilterNote("source", note.StatusDraft, "")
	source.ID = "20990101000000-1000"
	source.Links = []note.Link{{TargetID: "20990101000000-9999", Type: "supports", Annotation: "broken"}}
	writeNoteFile(t, nbDir, source)

	jsonOut, err := execute("graph", "show", "--focus", source.ID, "--format", "json")
	if err != nil {
		t.Fatalf("focused broken-edge JSON: %v", err)
	}
	var expected graphShowFilterResult
	if err := json.Unmarshal([]byte(jsonOut), &expected); err != nil {
		t.Fatalf("focused broken-edge JSON decode: %v", err)
	}
	mermaidOut, err := execute("graph", "show", "--focus", source.ID, "--format", "mermaid")
	if err != nil {
		t.Fatalf("focused broken-edge Mermaid: %v", err)
	}
	if len(expected.Edges) != 0 || strings.Contains(mermaidOut, "[missing]") || strings.Contains(mermaidOut, "-->") {
		t.Fatalf("focused broken-edge Mermaid diverged from filtered JSON: json=%s mermaid=%s", jsonOut, mermaidOut)
	}
}

func TestGraphShowMermaidPreservesBrokenEdgesWithPlaceholder(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	source := graphShowFilterNote("source", note.StatusDraft, "")
	source.ID = "20990101000000-1000"
	source.Links = []note.Link{{TargetID: "20990101000000-9999", Type: "supports", Annotation: "broken"}}
	writeNoteFile(t, nbDir, source)

	out, err := execute("graph", "show", "--format", "mermaid")
	if err != nil {
		t.Fatalf("broken-edge Mermaid: %v", err)
	}
	if !strings.Contains(out, `n1["20990101000000-9999  [missing]"]`) {
		t.Fatalf("broken-edge Mermaid missing placeholder: %s", out)
	}
	if !strings.Contains(out, `n0 -->|"supports — broken"| n1`) {
		t.Fatalf("broken-edge Mermaid missing stored edge: %s", out)
	}
}

func TestGraphShowMermaidNoFocusMatchesFullGraph(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	first := graphShowFilterNote("first", note.StatusDraft, "")
	second := graphShowFilterNote("second", note.StatusReviewed, "")
	first.ID = "20990101000000-1000"
	second.ID = "20990101000000-2000"
	first.Links = []note.Link{{TargetID: second.ID, Type: "supports"}}
	writeNoteFile(t, nbDir, first)
	writeNoteFile(t, nbDir, second)

	jsonOut, err := execute("graph", "show", "--format", "json")
	if err != nil {
		t.Fatalf("no-focus JSON: %v", err)
	}
	var expected graphShowFilterResult
	if err := json.Unmarshal([]byte(jsonOut), &expected); err != nil {
		t.Fatalf("no-focus JSON decode: %v", err)
	}
	mermaidOut, err := execute("graph", "show", "--format", "mermaid")
	if err != nil {
		t.Fatalf("no-focus Mermaid: %v", err)
	}
	var gotIDs []string
	for _, match := range mermaidNodePattern.FindAllStringSubmatch(mermaidOut, -1) {
		gotIDs = append(gotIDs, match[1])
	}
	var wantIDs []string
	aliasByID := map[string]string{}
	for i, n := range expected.Nodes {
		wantIDs = append(wantIDs, n.ID)
		aliasByID[n.ID] = "n" + strconv.Itoa(i)
	}
	var gotEdges []string
	for _, match := range mermaidEdgePattern.FindAllStringSubmatch(mermaidOut, -1) {
		gotEdges = append(gotEdges, match[1]+"->"+match[2])
	}
	var wantEdges []string
	for _, e := range expected.Edges {
		wantEdges = append(wantEdges, aliasByID[e.From]+"->"+aliasByID[e.To])
	}
	sort.Strings(gotIDs)
	sort.Strings(wantIDs)
	sort.Strings(gotEdges)
	sort.Strings(wantEdges)
	if strings.Join(gotIDs, ",") != strings.Join(wantIDs, ",") || strings.Join(gotEdges, ",") != strings.Join(wantEdges, ",") {
		t.Fatalf("no-focus Mermaid graph nodes=%v edges=%v, want nodes=%v edges=%v; output:\n%s", gotIDs, gotEdges, wantIDs, wantEdges, mermaidOut)
	}
}

func TestGraphShowMermaidInCommandHelp(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("graph", "show", "--help")
	if err != nil {
		t.Fatalf("graph show help: %v", err)
	}
	if !strings.Contains(out, "text, json, or mermaid") {
		t.Fatalf("graph show help missing Mermaid format: %s", out)
	}
}

func TestGraphShowMermaidDocumented(t *testing.T) {
	_, execute := setupNotebook(t)
	virtual, err := execute("show", "virtual-nn-cli-reference")
	if err != nil {
		t.Fatalf("virtual CLI reference: %v", err)
	}
	guide, err := os.ReadFile("../../../skills/nn-guide/SKILL.md")
	if err != nil {
		t.Fatalf("read nn-guide: %v", err)
	}
	for name, content := range map[string]string{"virtual CLI reference": virtual, "nn-guide": string(guide)} {
		for _, required := range []string{"--format text|json|mermaid", "constrain BFS expansion", "stored edge orientation"} {
			if !strings.Contains(content, required) {
				t.Fatalf("%s missing Mermaid graph-show contract %q", name, required)
			}
		}
	}
}

func TestGraphShowMermaidPreservesJSONOutput(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root := graphShowFilterNote("root", note.StatusDraft, "")
	child := graphShowFilterNote("child", note.StatusReviewed, "")
	root.ID = "20990101000000-1000"
	child.ID = "20990101000000-2000"
	root.Links = []note.Link{{TargetID: child.ID, Type: "supports", Annotation: "stable"}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	out, err := execute("graph", "show", "--focus", root.ID, "--depth", "1", "--format", "json")
	if err != nil {
		t.Fatalf("preserved graph JSON: %v", err)
	}
	want := "{\n  \"center\": \"20990101000000-1000\",\n  \"nodes\": [\n    {\n      \"id\": \"20990101000000-1000\",\n      \"title\": \"root\",\n      \"type\": \"concept\",\n      \"tags\": []\n    },\n    {\n      \"id\": \"20990101000000-2000\",\n      \"title\": \"child\",\n      \"type\": \"concept\",\n      \"tags\": []\n    }\n  ],\n  \"edges\": [\n    {\n      \"from\": \"20990101000000-1000\",\n      \"to\": \"20990101000000-2000\",\n      \"annotation\": \"stable\",\n      \"type\": \"supports\"\n    }\n  ]\n}\n"
	if out != want {
		t.Fatalf("graph JSON changed:\ngot=%q\nwant=%q", out, want)
	}
}

func TestGraphShowMermaidPreservesTextOutput(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root := graphShowFilterNote("root", note.StatusDraft, "")
	child := graphShowFilterNote("child", note.StatusReviewed, "")
	root.ID = "20990101000000-1000"
	child.ID = "20990101000000-2000"
	root.Links = []note.Link{{TargetID: child.ID, Type: "supports", Annotation: "stable"}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	out, err := execute("graph", "show", "--focus", root.ID, "--depth", "1", "--format", "text")
	if err != nil {
		t.Fatalf("preserved graph text: %v", err)
	}
	want := "20990101000000-1000  root\n  → [supports] 20990101000000-2000  child — stable\n"
	if out != want {
		t.Fatalf("graph text changed:\ngot=%q\nwant=%q", out, want)
	}
}

func TestGraphShowRejectsUnsupportedFormat(t *testing.T) {
	_, execute := setupNotebook(t)
	for _, format := range []string{"yaml", "MERMAID", ""} {
		_, err := execute("graph", "show", "--format", format)
		if err == nil {
			t.Fatalf("unsupported graph show format %q accepted", format)
		}
		if !strings.Contains(err.Error(), "unsupported format") {
			t.Fatalf("unsupported graph show format %q error = %q, want unsupported format", format, err)
		}
	}
}

func TestGraphShowMermaidAccepted(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root, _, _ := makeLinkedNotes(t, nbDir)

	out, err := execute("graph", "show", "--focus", root.ID, "--depth", "2", "--format", "mermaid")
	if err != nil {
		t.Fatalf("graph show mermaid: %v", err)
	}
	if got := strings.Count(out, "flowchart TD"); got != 1 {
		t.Fatalf("Mermaid flowchart declarations = %d, want 1; output:\n%s", got, out)
	}
}

var mermaidNodePattern = regexp.MustCompile(`(?m)^  n[0-9]+\["([^ ]+)  [^"]*"\]$`)

func TestGraphShowMermaidNodesMatchFilteredTraversal(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root := graphShowFilterNote("root", note.StatusDraft, "")
	outgoing := graphShowFilterNote("outgoing", note.StatusReviewed, "taxonomy")
	blocked := graphShowFilterNote("blocked", note.StatusDraft, "ontology")
	incoming := graphShowFilterNote("incoming", note.StatusPermanent, "taxonomy")
	root.Links = []note.Link{{TargetID: outgoing.ID, Type: "supports"}, {TargetID: blocked.ID, Type: "refines"}}
	incoming.Links = []note.Link{{TargetID: root.ID, Type: "governs"}}
	for _, n := range []*note.Note{root, outgoing, blocked, incoming} {
		writeNoteFile(t, nbDir, n)
	}

	cases := [][]string{
		{"--direction", "outgoing"},
		{"--direction", "incoming"},
		{"--direction", "both"},
		{"--links", "supports"},
		{"--status", "reviewed,permanent"},
		{"--representation", "taxonomy"},
		{"--direction", "both", "--links", "supports,governs", "--status", "reviewed,permanent", "--representation", "taxonomy"},
	}
	for _, extra := range cases {
		base := []string{"graph", "show", "--focus", root.ID, "--depth", "2"}
		jsonArgs := append(append([]string{}, base...), "--format", "json")
		jsonOut, err := execute(append(jsonArgs, extra...)...)
		if err != nil {
			t.Fatalf("filtered JSON %v: %v", extra, err)
		}
		var expected graphShowFilterResult
		if err := json.Unmarshal([]byte(jsonOut), &expected); err != nil {
			t.Fatalf("filtered JSON decode %v: %v", extra, err)
		}
		mermaidArgs := append(append([]string{}, base...), "--format", "mermaid")
		mermaidOut, err := execute(append(mermaidArgs, extra...)...)
		if err != nil {
			t.Fatalf("filtered Mermaid %v: %v", extra, err)
		}
		var wantIDs []string
		for _, n := range expected.Nodes {
			wantIDs = append(wantIDs, n.ID)
		}
		var gotIDs []string
		for _, match := range mermaidNodePattern.FindAllStringSubmatch(mermaidOut, -1) {
			gotIDs = append(gotIDs, match[1])
		}
		sort.Strings(wantIDs)
		sort.Strings(gotIDs)
		if strings.Join(gotIDs, ",") != strings.Join(wantIDs, ",") {
			t.Fatalf("filtered Mermaid nodes %v = %v, want %v; output:\n%s", extra, gotIDs, wantIDs, mermaidOut)
		}
	}
}

var mermaidEdgePattern = regexp.MustCompile(`(?m)^  (n[0-9]+) -->\|[^|]*\| (n[0-9]+)$`)

func TestGraphShowMermaidProvenanceDeduplicatesCSVFilters(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root := graphShowFilterNote("root", note.StatusDraft, "")
	child := graphShowFilterNote("child", note.StatusReviewed, "")
	root.Links = []note.Link{{TargetID: child.ID, Type: "supports"}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	out, err := execute("graph", "show", "--focus", root.ID, "--format", "mermaid", "--links", "supports,supports", "--status", "reviewed,reviewed")
	if err != nil {
		t.Fatalf("deduplicated Mermaid provenance: %v", err)
	}
	firstLine := strings.SplitN(out, "\n", 2)[0]
	if strings.Contains(firstLine, "supports,supports") || strings.Contains(firstLine, "reviewed,reviewed") {
		t.Fatalf("Mermaid provenance retained duplicate filters: %s", firstLine)
	}
}

func TestGraphShowMermaidProvenanceMetadata(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root := graphShowFilterNote("root", note.StatusDraft, "")
	child := graphShowFilterNote("child", note.StatusPermanent, "taxonomy")
	root.ID = "20990101000000-1000"
	child.ID = "20990101000000-2000"
	root.Links = []note.Link{{TargetID: child.ID, Type: "supports"}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	out, err := execute("graph", "show", "--focus", root.ID, "--depth", "3", "--direction", "both", "--links", "supports,governs", "--status", "permanent,reviewed", "--representation", "taxonomy", "--format", "mermaid")
	if err != nil {
		t.Fatalf("Mermaid provenance: %v", err)
	}
	want := "%% nn graph show focus=20990101000000-1000 depth=3 direction=both links=governs,supports status=permanent,reviewed representation=taxonomy"
	if !strings.Contains(out, want) {
		t.Fatalf("Mermaid provenance missing %q; output:\n%s", want, out)
	}
	for _, unstable := range []string{"generated_at=", "timestamp=", " time="} {
		if strings.Contains(out, unstable) {
			t.Fatalf("Mermaid provenance contains clock-derived field %q: %s", unstable, out)
		}
	}
}

func TestGraphShowMermaidDeterministic(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root := graphShowFilterNote("root", note.StatusDraft, "taxonomy")
	child := graphShowFilterNote("child", note.StatusReviewed, "taxonomy")
	root.ID = "20990101000000-1000"
	child.ID = "20990101000000-2000"
	root.Links = []note.Link{{TargetID: child.ID, Type: "supports", Annotation: "stable"}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)
	args := []string{"graph", "show", "--focus", root.ID, "--depth", "1", "--format", "mermaid", "--links", "supports", "--status", "reviewed", "--representation", "taxonomy"}

	baseline, err := execute(args...)
	if err != nil {
		t.Fatalf("deterministic Mermaid baseline: %v", err)
	}
	for i := 0; i < 10; i++ {
		got, err := execute(args...)
		if err != nil {
			t.Fatalf("deterministic Mermaid repeat: %v", err)
		}
		if got != baseline {
			t.Fatalf("Mermaid output changed between runs:\nfirst=%s\nrepeat=%s", baseline, got)
		}
	}
}

func TestGraphShowMermaidEscapesNodeLabels(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root := graphShowFilterNote("quote \" & <tag>\nline\rback\\slash", note.StatusDraft, "")
	root.ID = "20990101000000-1234"
	writeNoteFile(t, nbDir, root)

	out, err := execute("graph", "show", "--focus", root.ID, "--format", "mermaid")
	if err != nil {
		t.Fatalf("escaped Mermaid node: %v", err)
	}
	want := `n0["20990101000000-1234  quote #34; #38; #60;tag#62;#10;line#13;back#92;slash"]`
	if !strings.Contains(out, want) {
		t.Fatalf("escaped Mermaid node missing %q; output:\n%s", want, out)
	}
}

func TestGraphShowMermaidEscapesEdgeLabels(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root := graphShowFilterNote("root", note.StatusDraft, "")
	child := graphShowFilterNote("child", note.StatusReviewed, "")
	root.ID = "20990101000000-1000"
	child.ID = "20990101000000-2000"
	root.Links = []note.Link{{TargetID: child.ID, Type: "supports", Annotation: "pipe | quote \" & <tag> back\\slash"}}
	writeNoteFile(t, nbDir, root)
	writeNoteFile(t, nbDir, child)

	out, err := execute("graph", "show", "--focus", root.ID, "--format", "mermaid")
	if err != nil {
		t.Fatalf("escaped Mermaid edge: %v", err)
	}
	want := `n0 -->|"supports — pipe #124; quote #34; #38; #60;tag#62; back#92;slash"| n1`
	if !strings.Contains(out, want) {
		t.Fatalf("escaped Mermaid edge missing %q; output:\n%s", want, out)
	}
	if got := escapeMermaidLabel("\n\r"); got != "#10;#13;" {
		t.Fatalf("Mermaid control-character escaping = %q, want #10;#13;", got)
	}
}

func TestGraphShowMermaidEdgesPreserveStoredOrientation(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	root := graphShowFilterNote("root", note.StatusDraft, "")
	outgoing := graphShowFilterNote("outgoing", note.StatusReviewed, "")
	incoming := graphShowFilterNote("incoming", note.StatusReviewed, "")
	root.Links = []note.Link{{TargetID: outgoing.ID, Type: "supports"}}
	incoming.Links = []note.Link{{TargetID: root.ID, Type: "governs"}}
	for _, n := range []*note.Note{root, outgoing, incoming} {
		writeNoteFile(t, nbDir, n)
	}

	cases := [][]string{
		{"--direction", "outgoing"},
		{"--direction", "incoming"},
		{"--direction", "both"},
		{"--direction", "both", "--links", "supports"},
	}
	for _, extra := range cases {
		base := []string{"graph", "show", "--focus", root.ID, "--depth", "1"}
		jsonArgs := append(append([]string{}, base...), "--format", "json")
		jsonOut, err := execute(append(jsonArgs, extra...)...)
		if err != nil {
			t.Fatalf("edge JSON %v: %v", extra, err)
		}
		var expected graphShowFilterResult
		if err := json.Unmarshal([]byte(jsonOut), &expected); err != nil {
			t.Fatalf("edge JSON decode %v: %v", extra, err)
		}
		mermaidArgs := append(append([]string{}, base...), "--format", "mermaid")
		mermaidOut, err := execute(append(mermaidArgs, extra...)...)
		if err != nil {
			t.Fatalf("edge Mermaid %v: %v", extra, err)
		}
		aliasByID := map[string]string{}
		for i, n := range expected.Nodes {
			aliasByID[n.ID] = "n" + strconv.Itoa(i)
		}
		var wantPairs []string
		for _, e := range expected.Edges {
			wantPairs = append(wantPairs, aliasByID[e.From]+"->"+aliasByID[e.To])
		}
		var gotPairs []string
		for _, match := range mermaidEdgePattern.FindAllStringSubmatch(mermaidOut, -1) {
			gotPairs = append(gotPairs, match[1]+"->"+match[2])
		}
		sort.Strings(wantPairs)
		sort.Strings(gotPairs)
		if strings.Join(gotPairs, ",") != strings.Join(wantPairs, ",") {
			t.Fatalf("Mermaid oriented edges %v = %v, want %v; output:\n%s", extra, gotPairs, wantPairs, mermaidOut)
		}
	}
}
