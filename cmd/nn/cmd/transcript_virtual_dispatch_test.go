package cmd

import (
	"strings"
	"testing"
)

func TestGlobalProtocolDispatchesTranscriptSkill(t *testing.T) {
	_, execute := setupNotebook(t)
	out, e := execute("show", "--global")
	if e != nil {
		t.Fatal(e)
	}
	for _, required := range []string{"Transcript exploration / Transcript Office", "nn skills get nn-transcript", "nn skills get nn-transcript --reference <name>", "Before notebook-graph navigation", "agent runs", "assignments", "execution-pattern discovery"} {
		if !strings.Contains(out, required) {
			t.Errorf("ASSERT_GLOBAL_TRANSCRIPT_DISPATCH: missing %q", required)
		}
	}
	if strings.Contains(out, "Before human-driven iterative navigation — including") {
		t.Fatal("ASSERT_GLOBAL_GRAPH_SCOPE: unqualified navigation dispatch remains")
	}
	list, e := execute("skills", "list")
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(list, "nn-transcript") {
		t.Fatal("ASSERT_GLOBAL_TRANSCRIPT_DISPATCH: dispatched skill not listed")
	}
	core, e := execute("skills", "get", "nn-transcript")
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(core, "Transcript Office") {
		t.Fatal("ASSERT_GLOBAL_TRANSCRIPT_DISPATCH: skill not retrievable")
	}
}
