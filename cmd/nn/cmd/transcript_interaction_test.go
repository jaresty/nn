package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

// These guards validate the shipped instruction artifact, not LLM behavior.
func TestTranscriptInteractionContract(t *testing.T) {
	_, execute := setupNotebook(t)
	body, err := execute("skills", "get", "nn-transcript", "--reference", "interaction")
	if err != nil {
		t.Fatalf("ASSERT_INTERACTION_OWNER: shared interaction contract unavailable: %v", err)
	}
	for _, term := range []string{"explicit operand", "displayed matching action", "selected target", "navigation state", "previous_view", "scope definition", "follow-up", "budget", "compaction", "exact restoration unavailable", "empty filter", "capture", "no notebook write", "--format text", "--event", "--payload"} {
		if !strings.Contains(body, term) {
			t.Errorf("ASSERT_INTERACTION_CONTRACT: missing %q", term)
		}
	}
	for _, owner := range []string{"actions", "discovery", "navigate", "rooms", "review", "lenses"} {
		text, e := execute("skills", "get", "nn-transcript", "--reference", owner)
		if e != nil {
			t.Fatal(e)
		}
		if !strings.Contains(text, "--reference interaction") {
			t.Errorf("ASSERT_INTERACTION_DISPATCH: %s does not dispatch shared owner", owner)
		}
	}
	core, e := execute("skills", "get", "nn-transcript")
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(core, "--reference interaction") {
		t.Fatal("ASSERT_INTERACTION_DISPATCH: core lacks interaction owner")
	}
}

// Native baseline capability test, not a claim that a model obeys navigation rules.
// Existing commands must support a readable suggestion followed by exact evidence.
func TestTranscriptInteractionNativeBaseline(t *testing.T) {
	_, execute := setupNotebook(t)
	path := reviewFixture(t)
	text, err := execute("transcript", "review", path, "--order", "canonical", "--limit", "1", "--last", "2", "--format", "text")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "failed") {
		t.Fatal("ASSERT_INTERACTION_BASELINE: readable failure missing")
	}
	first, e := buildReviewTails(path, "open-handoff", "canonical", "", 1, "", 2, true, 1, "")
	if e != nil {
		t.Fatal(e)
	}
	var id, agent string
	for _, raw := range first.Events {
		var ev map[string]json.RawMessage
		if e = json.Unmarshal(raw, &ev); e != nil {
			t.Fatal(e)
		}
		if ledgerString(ev, "kind") == "tool_result" {
			id = ledgerString(ev, "event_id")
			agent = ledgerString(ev, "agent_id")
			break
		}
	}
	if id == "" {
		t.Fatal("ASSERT_INTERACTION_BASELINE: fixture lacks nominated result")
	}
	exact, e := execute("transcript", "events", path, agent, "--event", id, "--payload", "--json")
	if e != nil {
		t.Fatal(e)
	}
	var p ledgerPage
	if e = json.Unmarshal([]byte(exact), &p); e != nil {
		t.Fatal(e)
	}
	if p.Pages != 1 || len(p.Events) != 1 || !strings.Contains(string(p.Events[0]), id) || !strings.Contains(string(p.Events[0]), "failed") {
		t.Fatal("ASSERT_INTERACTION_BASELINE: exact inspection changed target or lost evidence")
	}
}

// Guard explicit commitments and reject the exact conflicting directives found
// in the ADR review. Presence alone is not a semantic model-compliance test.
func TestTranscriptInteractionSingleOwnerRules(t *testing.T) {
	_, execute := setupNotebook(t)
	cases := []struct{ name, owner, required, forbidden string }{
		{"TargetPromise", "interaction", "2. A uniquely displayed matching action wins over a background selected target.", "overrides older fallback-menu examples"},
		{"BudgetMonotonic", "interaction", "The inspection budget ledger is monotonic across Back", "Back restores the unused budget"},
		{"BackReplay", "interaction", "**Back restores navigation state, not identical prose.**", "write numbered JSON view records"},
		{"NoManualPersistence", "interaction", "Do not create temporary files or serialize view JSON for navigation.", "`mktemp -d`"},
		{"NoDeterministicProse", "interaction", "LLM rerendering is not deterministic", "exact rendered response"},
		{"ScopeRestriction", "interaction", "Respect a concrete user restriction:", "Target resolution is not acquisition permission."},
		{"RoomEntry", "rooms", "--last 5 --include-assignment --format text --max-text-chars 1000", "Retrieve a metadata-oriented bounded tail"},
		{"ActionCount", "actions", "Promote up to three useful next actions", "Promote two or three useful next actions"},
		{"BoundComparison", "lenses", "Operands already bound", "never authorize the LLM to supply missing"},
		{"FindIntent", "review", "Find is an intent, not a compulsory filter chooser.", "Find always opens the filter chooser"},
		{"CaptureApproval", "actions", "Require explicit approval of the concrete proposal before creating/updating a note or link", "Capture this insight writes immediately"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body, e := execute("skills", "get", "nn-transcript", "--reference", tc.owner)
			if e != nil {
				t.Fatal(e)
			}
			if !strings.Contains(body, tc.required) || strings.Contains(body, tc.forbidden) {
				t.Fatalf("ASSERT_SINGLE_OWNER_%s: contradictory or missing commitment", tc.name)
			}
		})
	}
	core, e := execute("skills", "get", "nn-transcript")
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(core, "On `:enter`, show 2–4 salient findings") || strings.Contains(core, "Behavioral patterns require\nreading the selected whole sessions") {
		t.Fatal("ASSERT_SINGLE_OWNER_CORE: unconditional entry/whole-session rule remains")
	}
}
