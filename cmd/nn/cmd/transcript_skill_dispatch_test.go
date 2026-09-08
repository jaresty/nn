package cmd

import (
	"strings"
	"testing"
)

func TestTranscriptSkillDescriptionLookupRouting(t *testing.T) {
	const a = "ASSERT_TRANSCRIPT_DESCRIPTION_LOOKUP_ROUTING"
	_, execute := setupNotebook(t)
	core, err := execute("skills", "get", "nn-transcript")
	if err != nil {
		t.Fatal(err)
	}
	discovery, err := execute("skills", "get", "nn-transcript", "--reference", "discovery")
	if err != nil {
		t.Fatal(err)
	}
	handoffs, err := execute("skills", "get", "nn-transcript", "--reference", "handoffs")
	if err != nil {
		t.Fatal(err)
	}
	for name, text := range map[string]string{"core": core, "discovery": discovery, "handoffs": handoffs} {
		for _, phrase := range []string{"launch name", "tree --json", "description", "do not use `nn transcript search`"} {
			if !strings.Contains(text, phrase) {
				t.Fatalf("%s: %s lacks %q", a, name, phrase)
			}
		}
	}
	if !strings.Contains(core, "`nn transcript ls`") || !strings.Contains(discovery, "select the parent session") {
		t.Fatalf("%s: ls to parent-selection route absent", a)
	}
	t.Log(a + ": PASS")
}

func TestTranscriptSkillLazyDispatch(t *testing.T) {
	const a = "ASSERT_TRANSCRIPT_LAZY_DISPATCH"
	_, execute := setupNotebook(t)
	core, err := execute("skills", "get", "nn-transcript")
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(core, "\n"); lines > 200 || len(core) > 14000 {
		t.Fatalf("%s: core is not compact: %d lines, %d bytes", a, lines, len(core))
	}
	listing, err := execute("skills", "get", "nn-transcript", "--list-references")
	if err != nil {
		t.Fatal(err)
	}
	for name, terms := range map[string][]string{
		"discovery": {"summary.cost", "--cursor", "topology_status"},
		"events":    {"--payload", "--snapshot", "--errors-only", "--since", "--until", "UNBOUNDED"},
		"summaries": {"--summary usage", "--summary tools", "--summary timing", "not execution time"},
		"handoffs":  {"--at launch", "--at return", "description", "occurrence", "last_terminal_record"},
		"navigate":  {"semantic thread-layout contract", "not independently verified"},
		"patterns":  {"whole sessions", "four assertions"},
	} {
		if !strings.Contains(core, "--reference "+name) || !strings.Contains(listing, name) {
			t.Fatalf("%s: %s not dispatched/listed", a, name)
		}
		text, err := execute("skills", "get", "nn-transcript", "--reference", name)
		if err != nil {
			t.Fatalf("%s: %s not retrievable: %v", a, name, err)
		}
		for _, term := range terms {
			if !strings.Contains(text, term) {
				t.Fatalf("%s: %s lacks %s", a, name, term)
			}
		}
	}
	t.Log(a + ": PASS")
}
