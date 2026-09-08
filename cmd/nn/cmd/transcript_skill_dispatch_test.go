package cmd

import (
	"os"
	"path/filepath"
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
		for _, phrase := range []string{"launch name", "description", "do not use `nn transcript search`"} {
			if !strings.Contains(text, phrase) {
				t.Fatalf("%s: %s lacks %q", a, name, phrase)
			}
		}
	}
	if !strings.Contains(discovery, "tree <session> --description") || !strings.Contains(discovery, "every exact case-sensitive match") {
		t.Fatalf("%s: discovery lacks native exact description selector contract", a)
	}
	if !strings.Contains(core, "`nn transcript ls`") || !strings.Contains(discovery, "select the parent session") {
		t.Fatalf("%s: ls to parent-selection route absent", a)
	}
	t.Log(a + ": PASS")
}

func TestTranscriptOfficeDefaultAndLensDispatch(t *testing.T) {
	root := filepath.Join("..", "..", "..", "skills", "nn-transcript")
	read := func(name string) string {
		t.Helper()
		body, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			return ""
		}
		return string(body)
	}
	assertContains := func(assertion, text string, required ...string) {
		t.Helper()
		for _, phrase := range required {
			if !strings.Contains(text, phrase) {
				t.Errorf("%s: missing %q", assertion, phrase)
			}
		}
	}

	core := read("SKILL.md")
	navigate := read(filepath.Join("references", "navigate.md"))
	rooms := read(filepath.Join("references", "rooms.md"))
	lenses := read(filepath.Join("references", "lenses.md"))

	assertContains("ASSERT_TRANSCRIPT_OFFICE_DEFAULT_ENTRY", core,
		"default entry experience", "Transcript Office", "--reference navigate")
	assertContains("ASSERT_TRANSCRIPT_OFFICE_LLM_MEDIATED_NO_CLI_PICKER", core,
		"LLM-mediated", "not an interactive CLI")
	assertContains("ASSERT_TRANSCRIPT_OFFICE_AUTHENTICATED_TOPOLOGY", navigate,
		"authenticated topology", "direct children", "nested manager")
	assertContains("ASSERT_TRANSCRIPT_ROOM_OPEN_LENS", rooms,
		"Situation Board", "user-defined", "named lenses are presets")
	assertContains("ASSERT_TRANSCRIPT_OFFICE_LENS_SCAN", lenses,
		"Office Scan", "authenticated population", "uninspected")
	assertContains("ASSERT_TRANSCRIPT_LENS_DRILLDOWN_PRESERVES_QUESTION", lenses,
		"preserves the question", "Back", "same Office Scan")
	assertContains("ASSERT_TRANSCRIPT_EVIDENCE_QUALIFICATION", lenses,
		"agent report", "inspected retained result", "independent verification")
	assertContains("ASSERT_TRANSCRIPT_NO_INFERRED_LIVE_STATUS", core,
		"not proof of current activity", "Missing return does not prove running")
	assertContains("ASSERT_TRANSCRIPT_REFRESH_STATE", rooms,
		"retains the lens", "reacquires", "retained snapshot")
	assertContains("ASSERT_TRANSCRIPT_ROOM_BOUNDED_EXPANSION", rooms,
		"--last 5", "--last 20", "replacement snapshot", "not stable backward continuation")
	assertContains("ASSERT_TRANSCRIPT_OFFICE_LAZY_REFERENCES", core,
		"--reference rooms", "--reference lenses")
}

func TestTranscriptOfficeParentageAndLifecycleAuthority(t *testing.T) {
	const a = "ASSERT_TRANSCRIPT_OFFICE_PARENTAGE_LIFECYCLE_AUTHORITY"
	skillDir := filepath.Join("..", "..", "..", "skills", "nn-transcript")
	data, err := os.ReadFile(filepath.Join(skillDir, "references", "navigate.md"))
	if err != nil {
		t.Fatalf("%s: read navigate: %v", a, err)
	}
	text := string(data)
	for _, phrase := range []string{
		"parentage_status", "`recorded`", "`conservative_root`", "`repaired`",
		"Only recorded edges may be called authenticated",
		"No nested parent edges were recovered in this selected transcript",
		"RETURN OBSERVED", "TERMINAL OBSERVED", "PROVISIONAL", "INTERRUPTED", "UNAVAILABLE",
		"These are evidence lanes, never alive/dead labels",
		"A background launch is launch mode rather than current activity",
		"task success",
	} {
		if !strings.Contains(text, phrase) {
			t.Errorf("%s: missing %q", a, phrase)
		}
	}
}

func TestTranscriptOfficeReadableLobbyLabels(t *testing.T) {
	const a = "ASSERT_TRANSCRIPT_OFFICE_READABLE_LOBBY_LABELS"
	root := filepath.Join("..", "..", "..", "skills", "nn-transcript")
	for file, required := range map[string][]string{
		"SKILL.md": {"readable `label`", "opening_label", "label_provenance", "session ID", "recorded", "recent", "opening", "interpreted", "untitled", "conversation_kind", "owner_session", "open_window_status"},
		filepath.Join("references", "discovery.md"): {
			"opening_label", "latest non-acknowledgement ROOT user message", "label_provenance", "recent", "opening", "interpreted", "untitled", "recorded", "exact session ID",
			"retain the complete selected row across Back", "Reacquire labels only on explicit discovery refresh",
			"conversation_kind", "owner_session", "cross-session ownership is authenticated",
			"open_window_status", "retained transcripts do not establish which Pi windows are open",
		},
	} {
		body, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Errorf("%s: cannot read %s: %v", a, file, err)
			continue
		}
		for _, phrase := range required {
			if !strings.Contains(string(body), phrase) {
				t.Errorf("%s: %s missing %q", a, file, phrase)
			}
		}
	}
}

func TestTranscriptOfficeTargetFirstDiscovery(t *testing.T) {
	const a = "ASSERT_TRANSCRIPT_OFFICE_TARGET_FIRST_DISCOVERY"
	root := filepath.Join("..", "..", "..", "skills", "nn-transcript")
	for file, required := range map[string][]string{
		"SKILL.md": {"explicit project", "before generic recent-session discovery"},
		filepath.Join("references", "discovery.md"): {"target hint", "project, workspace, or office", "targeted scope first", "ambiguous"},
	} {
		body, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatalf("%s: %v", a, err)
		}
		for _, phrase := range required {
			if !strings.Contains(string(body), phrase) {
				t.Errorf("%s: %s missing %q", a, file, phrase)
			}
		}
	}
}

func TestTranscriptOfficeCanonicalPathCustody(t *testing.T) {
	const a = "ASSERT_TRANSCRIPT_OFFICE_CANONICAL_PATH_CUSTODY"
	root := filepath.Join("..", "..", "..", "skills", "nn-transcript")
	for file, required := range map[string][]string{
		"SKILL.md": {"exact `path`", "never reconstruct"},
		filepath.Join("references", "discovery.md"): {"selected `ls` row", "byte-for-byte", "tree", "events", "show"},
		filepath.Join("references", "navigate.md"):  {"canonical path", "actual error", "scoped cohort"},
	} {
		body, err := os.ReadFile(filepath.Join(root, file))
		if err != nil {
			t.Fatalf("%s: %v", a, err)
		}
		for _, phrase := range required {
			if !strings.Contains(string(body), phrase) {
				t.Errorf("%s: %s missing %q", a, file, phrase)
			}
		}
	}
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
		"navigate":  {"Transcript Office", "authenticated topology", "Situation Board"},
		"rooms":     {"Situation Board", "--last 5", "replacement snapshot"},
		"lenses":    {"Office Scan", "user-defined", "independent verification"},
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
