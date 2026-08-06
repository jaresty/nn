package cmd

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/jaresty/nn/internal/note"
)

// Assertion: text output includes governing protocols section when a governs-backlink exists.
func TestShowGoverningProtocolsText(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	proto := newTestNoteForCLI(note.GenerateID(), "My Protocol", note.TypeProtocol)
	target := newTestNoteForCLI(note.GenerateID(), "Target Note", note.TypeConcept)
	proto.Links = []note.Link{{TargetID: target.ID, Annotation: "governs this", Type: "governs"}}
	writeNoteFile(t, nbDir, proto)
	writeNoteFile(t, nbDir, target)

	out, err := execute("show", target.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if !strings.Contains(out, "governing protocols") {
		t.Errorf("expected 'governing protocols' section in output:\n%s", out)
	}
	if !strings.Contains(out, "My Protocol") {
		t.Errorf("expected protocol title 'My Protocol' in output:\n%s", out)
	}
}

// Assertion: text output omits governing protocols section when none exist.
func TestShowNoGoverningProtocolsText(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	target := newTestNoteForCLI(note.GenerateID(), "Ungoverned Note", note.TypeConcept)
	writeNoteFile(t, nbDir, target)

	out, err := execute("show", target.ID)
	if err != nil {
		t.Fatalf("nn show: %v", err)
	}
	if strings.Contains(out, "governing protocols") {
		t.Errorf("expected no 'governing protocols' section when none exist:\n%s", out)
	}
}

// Assertion: JSON output includes governing_protocols array with protocol ID and title.
func TestShowGoverningProtocolsJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	proto := newTestNoteForCLI(note.GenerateID(), "JSON Protocol", note.TypeProtocol)
	target := newTestNoteForCLI(note.GenerateID(), "JSON Target", note.TypeConcept)
	proto.Links = []note.Link{{TargetID: target.ID, Annotation: "governs this", Type: "governs"}}
	writeNoteFile(t, nbDir, proto)
	writeNoteFile(t, nbDir, target)

	out, err := execute("show", "--json", target.ID)
	if err != nil {
		t.Fatalf("nn show --json: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	gp, ok := result["governing_protocols"]
	if !ok {
		t.Fatalf("JSON missing 'governing_protocols' key:\n%s", out)
	}
	arr, ok := gp.([]any)
	if !ok || len(arr) == 0 {
		t.Fatalf("expected non-empty governing_protocols array, got: %v", gp)
	}
	entry, ok := arr[0].(map[string]any)
	if !ok {
		t.Fatalf("governing_protocols entry is not an object: %v", arr[0])
	}
	if entry["title"] != "JSON Protocol" {
		t.Errorf("expected title 'JSON Protocol', got %v", entry["title"])
	}
}

// Assertion: JSON output governing_protocols is empty array when none exist.
func TestShowNoGoverningProtocolsJSON(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	target := newTestNoteForCLI(note.GenerateID(), "Ungoverned JSON Note", note.TypeConcept)
	writeNoteFile(t, nbDir, target)

	out, err := execute("show", "--json", target.ID)
	if err != nil {
		t.Fatalf("nn show --json: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	gp, ok := result["governing_protocols"]
	if !ok {
		t.Fatalf("JSON missing 'governing_protocols' key:\n%s", out)
	}
	arr, ok := gp.([]any)
	if !ok {
		t.Fatalf("governing_protocols should be an array, got: %T", gp)
	}
	if len(arr) != 0 {
		t.Errorf("expected empty governing_protocols array, got: %v", arr)
	}
}

// Assertion: governing protocols include a uniquely resolved representation root only.
func TestFindGoverningProtocolsIncludesUniqueRepresentationRoot(t *testing.T) {
	root := newTestNoteForCLI("root", "Root", note.TypeModel)
	root.Representation = "ontology"
	child := newTestNoteForCLI("child", "Child", note.TypeConcept)
	child.Representation = "ontology"
	root.Links = []note.Link{{TargetID: child.ID, Type: "extends", Annotation: "contains child"}}

	direct := newTestNoteForCLI("proto-a", "Direct", note.TypeProtocol)
	direct.Links = []note.Link{{TargetID: child.ID, Type: "governs", Annotation: "direct"}}
	inherited := newTestNoteForCLI("proto-b", "Inherited", note.TypeProtocol)
	inherited.Links = []note.Link{{TargetID: root.ID, Type: "governs", Annotation: "root"}}
	both := newTestNoteForCLI("proto-c", "Both", note.TypeProtocol)
	both.Links = []note.Link{
		{TargetID: child.ID, Type: "governs", Annotation: "direct"},
		{TargetID: root.ID, Type: "governs", Annotation: "root"},
	}

	plain := newTestNoteForCLI("plain", "Plain", note.TypeConcept)
	plainDirect := newTestNoteForCLI("proto-d", "Plain direct", note.TypeProtocol)
	plainDirect.Links = []note.Link{{TargetID: plain.ID, Type: "governs", Annotation: "plain"}}

	rootless := newTestNoteForCLI("rootless", "Rootless", note.TypeConcept)
	rootless.Representation = "ontology"
	rootlessDirect := newTestNoteForCLI("proto-e", "Rootless direct", note.TypeProtocol)
	rootlessDirect.Links = []note.Link{{TargetID: rootless.ID, Type: "governs", Annotation: "rootless"}}

	ambiguous := newTestNoteForCLI("ambiguous", "Ambiguous", note.TypeConcept)
	ambiguous.Representation = "ontology"
	parentA := newTestNoteForCLI("parent-a", "Parent A", note.TypeModel)
	parentA.Representation = "ontology"
	parentA.Links = []note.Link{{TargetID: ambiguous.ID, Type: "extends", Annotation: "parent A"}}
	parentB := newTestNoteForCLI("parent-b", "Parent B", note.TypeModel)
	parentB.Representation = "ontology"
	parentB.Links = []note.Link{{TargetID: ambiguous.ID, Type: "extends", Annotation: "parent B"}}
	ambiguousInherited := newTestNoteForCLI("proto-f", "Ambiguous inherited", note.TypeProtocol)
	ambiguousInherited.Links = []note.Link{{TargetID: parentA.ID, Type: "governs", Annotation: "ambiguous root"}}
	ambiguousDirect := newTestNoteForCLI("proto-g", "Ambiguous direct", note.TypeProtocol)
	ambiguousDirect.Links = []note.Link{{TargetID: ambiguous.ID, Type: "governs", Annotation: "direct"}}

	cycleA := newTestNoteForCLI("cycle-a", "Cycle A", note.TypeConcept)
	cycleA.Representation = "ontology"
	cycleB := newTestNoteForCLI("cycle-b", "Cycle B", note.TypeConcept)
	cycleB.Representation = "ontology"
	cycleA.Links = []note.Link{{TargetID: cycleB.ID, Type: "extends", Annotation: "cycle"}}
	cycleB.Links = []note.Link{{TargetID: cycleA.ID, Type: "extends", Annotation: "cycle"}}
	cycleDirect := newTestNoteForCLI("proto-h", "Cycle direct", note.TypeProtocol)
	cycleDirect.Links = []note.Link{{TargetID: cycleA.ID, Type: "governs", Annotation: "direct"}}

	modelCycleRoot := newTestNoteForCLI("model-cycle-root", "Model cycle root", note.TypeModel)
	modelCycleRoot.Representation = "ontology"
	modelCycleChild := newTestNoteForCLI("model-cycle-child", "Model cycle child", note.TypeConcept)
	modelCycleChild.Representation = "ontology"
	modelCycleRoot.Links = []note.Link{{TargetID: modelCycleChild.ID, Type: "extends", Annotation: "down"}}
	modelCycleChild.Links = []note.Link{{TargetID: modelCycleRoot.ID, Type: "extends", Annotation: "back"}}
	modelCycleInherited := newTestNoteForCLI("proto-i", "Model cycle inherited", note.TypeProtocol)
	modelCycleInherited.Links = []note.Link{{TargetID: modelCycleRoot.ID, Type: "governs", Annotation: "root"}}
	modelCycleDirect := newTestNoteForCLI("proto-j", "Model cycle direct", note.TypeProtocol)
	modelCycleDirect.Links = []note.Link{{TargetID: modelCycleChild.ID, Type: "governs", Annotation: "direct"}}

	ids := func(protocols []*note.Note) []string {
		result := make([]string, len(protocols))
		for i, protocol := range protocols {
			result[i] = protocol.ID
		}
		return result
	}
	got := map[string][]string{
		"unique root": ids(findGoverningProtocols(child.ID, []*note.Note{both, inherited, child, root, direct})),
		"plain":       ids(findGoverningProtocols(plain.ID, []*note.Note{root, inherited, plain, plainDirect})),
		"rootless":    ids(findGoverningProtocols(rootless.ID, []*note.Note{rootlessDirect, rootless})),
		"ambiguous":   ids(findGoverningProtocols(ambiguous.ID, []*note.Note{ambiguousInherited, parentA, ambiguousDirect, ambiguous, parentB})),
		"cycle":       ids(findGoverningProtocols(cycleA.ID, []*note.Note{cycleB, cycleDirect, cycleA})),
		"model cycle": ids(findGoverningProtocols(modelCycleChild.ID, []*note.Note{modelCycleRoot, modelCycleInherited, modelCycleChild, modelCycleDirect})),
	}
	want := map[string][]string{
		"unique root": {"proto-a", "proto-b", "proto-c"},
		"plain":       {"proto-d"},
		"rootless":    {"proto-e"},
		"ambiguous":   {"proto-g"},
		"cycle":       {"proto-h"},
		"model cycle": {"proto-j"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("governing protocols = %#v, want %#v", got, want)
	}
}

func TestShowLinkedFromGoverningProtocols(t *testing.T) {
	nbDir, execute := setupNotebook(t)
	proto := newTestNoteForCLI(note.GenerateID(), "Linked-From Protocol", note.TypeProtocol)
	src := newTestNoteForCLI(note.GenerateID(), "Source Note", note.TypeConcept)
	target := newTestNoteForCLI(note.GenerateID(), "Target Note", note.TypeConcept)
	src.Links = []note.Link{{TargetID: target.ID, Annotation: "links to", Type: "extends"}}
	proto.Links = []note.Link{{TargetID: target.ID, Annotation: "governs this", Type: "governs"}}
	writeNoteFile(t, nbDir, proto)
	writeNoteFile(t, nbDir, src)
	writeNoteFile(t, nbDir, target)

	out, err := execute("show", "--linked-from", src.ID)
	if err != nil {
		t.Fatalf("nn show --linked-from: %v", err)
	}
	if !strings.Contains(out, "governing protocols") {
		t.Errorf("expected 'governing protocols' in --linked-from output:\n%s", out)
	}
	if !strings.Contains(out, "Linked-From Protocol") {
		t.Errorf("expected protocol title in --linked-from output:\n%s", out)
	}
}
