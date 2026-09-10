package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

// Publication checks, not a semantic evaluator or an LLM-compliance test.
// Keep the exact requirement identity in output so isolated mutations are attributable.
func TestAttentionDiscoveryPublication(t *testing.T) {
	_, execute := setupNotebook(t)
	cases := []struct{ id, owner, required string }{
		{"AD1_SCOPE", "attention", "A request for attention across a named cohort retains that cohort, filters and canonical paths rather than substituting a remembered room."},
		{"AD2_RESTRICTION", "interaction", "Respect a concrete user restriction:"},
		{"AD3_DIRECT", "interaction", "Ordinary bounded read-only requests execute before optional choices."},
		{"AD4_ATTEMPTS", "attention", "Count attempted candidates when assignment/work acquisition starts, including errors and unknowns."},
		{"AD5_OUTCOMES", "attention", "known other task scope is **outside policy scope**."},
		{"AD6_REPLACEMENT", "attention", "For each selected candidate make one evaluation, with no automatic retry or replacement."},
		{"AD7_WIDENING", "attention", "An empty filter stays empty."},
		{"AD8_BACK", "interaction", "Back does not replenish already-spent reads."},
		{"AD9_AUTOMATION", "interaction", "There is no background monitor or mandatory periodic schedule."},
		{"AD10_EVALUATOR", "attention", "nn's existing Datalog parser/evaluator"},
		{"AD11_IMPERATIVE", "attention", "Execute before optional actions."},
		{"AD12_RECOVERY", "attention", "“Check if the signal has recovered” requests a fresh bounded check of the identified stream, not a confirmation proposal."},
		{"AD13_NO_CONFIRM", "interaction", "not simply because another read follows."},
	}
	normalize := func(s string) string { return strings.Join(strings.Fields(s), " ") }
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			t.Logf("procedure: TestAttentionDiscoveryPublication/%s; assertion: %s", tc.id, tc.id)
			text, err := execute("skills", "get", "nn-transcript", "--reference", tc.owner)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(normalize(text), tc.required) {
				t.Fatalf("%s FAIL: required publication clause absent", tc.id)
			}
			t.Logf("%s PASS", tc.id)
		})
	}
}

// Publication only: these do not emulate an LLM or introduce a runtime hook.
func TestAttentionDirectReadAndRetention(t *testing.T) {
	_, execute := setupNotebook(t)
	for _, clause := range []string{
		"Replay and inspection do not reopen sources or re-evaluate policy.",
		"Optional replay task/agent flags must match the retained selection.",
		"Explicit Refresh evaluates anew within current scope and actual limits.",
		"capture still needs a concrete approved proposal.",
		"No notebook policy activation, standing-approval wizard, background monitoring, timers or new engine is involved.",
	} {
		t.Run(clause, func(t *testing.T) {
			text, err := execute("skills", "get", "nn-transcript", "--reference", "attention")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(strings.Join(strings.Fields(text), " "), clause) {
				t.Fatalf("STANDING FAIL: %s", clause)
			}
			t.Logf("STANDING PASS: %s", clause)
		})
	}
	for _, owner := range []string{"discovery", "navigate", "review", "rooms", "interaction"} {
		t.Run(owner, func(t *testing.T) {
			text, err := execute("skills", "get", "nn-transcript", "--reference", owner)
			if err != nil {
				t.Fatal(err)
			}
			for _, stale := range []string{"apply standing attention", "offer its one-time opt-in", "Without standing approval, entering a view"} {
				if strings.Contains(text, stale) {
					t.Fatalf("TRACER_NO_STANDING FAIL: %s retains %q", owner, stale)
				}
			}
		})
	}
}

func TestTranscriptObservationBeforePicker(t *testing.T) {
	_, execute := setupNotebook(t)
	core, _ := execute("skills", "get", "nn-transcript")
	text, err := execute("skills", "get", "nn-transcript", "--reference", "discovery")
	if err != nil {
		t.Fatal(err)
	}
	for _, clause := range []string{"--reference observe", "before choices", "Selecting a conversation opens", "explicit questions bypass browsing", "More conversations…"} {
		if !strings.Contains(text, clause) {
			t.Fatalf("PICKER FAIL: missing %s", clause)
		}
	}
	if !strings.Contains(core, "**Bare invocation / what is happening:**") {
		t.Fatal("PICKER FAIL: core lacks default dispatch")
	}
	attention, _ := execute("skills", "get", "nn-transcript", "--reference", "attention")
	for _, stale := range []string{"offer three passes", "remaining passes", "total pass count"} {
		if strings.Contains(attention, stale) {
			t.Fatalf("STANDING EXPIRY FAIL: %s", stale)
		}
	}
}

// Command compatibility only: this does not implement or evaluate LLM task classification.
func TestAttentionDiscoveryNativeRecipe(t *testing.T) {
	_, execute := setupNotebook(t)
	path := reviewFixture(t)
	out, err := execute("transcript", "review", path, "--queue", "archive", "--order", "observed-recent", "--limit", "1", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var metadata struct {
		Rows []struct {
			ID string `json:"id"`
		} `json:"rows"`
	}
	if err = json.Unmarshal([]byte(out), &metadata); err != nil || len(metadata.Rows) != 1 {
		t.Fatalf("native selection failed: %s %v", out, err)
	}
	id := metadata.Rows[0].ID
	out, err = execute("transcript", "context", path, id, "--last", "1", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var contextPage ledgerPage
	if err = json.Unmarshal([]byte(out), &contextPage); err != nil || contextPage.Pages != 1 || contextPage.Snapshot == "" {
		t.Fatalf("native context did not fit fixture reservation: %s %v", out, err)
	}
	// Scope is explicit test input, not inferred from the fixture's launch label.
	for _, task := range []string{"implementation", "", "research"} {
		out, err = execute("transcript", "attention", path, "--agent", id, "--task", task, "--format", "json")
		if err != nil {
			t.Fatal(err)
		}
		var p attentionPage
		if err = json.Unmarshal([]byte(out), &p); err != nil || len(p.Rooms) != 1 || p.Rooms[0].ID != id || p.Task != task {
			t.Fatalf("native evaluation changed identity/scope: %s %v", out, err)
		}
		if len(p.Rooms[0].Signals) != 1 {
			t.Fatal("missing per-signal result")
		}
		s := p.Rooms[0].Signals[0]
		if task == "research" && s.Outcome != "not_applicable" {
			t.Fatal("known other task became applicable")
		}
		if task == "" && (s.Applicability.Status != "unknown" || s.Condition.Status == "inapplicable") {
			t.Fatal("unknown task suppressed measurement")
		}
	}
	t.Log("NATIVE_RECIPE_PASS: one metadata page, one context page, exact selected room; versioned applicability remains separate from measurement")
}

func TestAttentionDiscoveryDispatch(t *testing.T) {
	_, execute := setupNotebook(t)
	for _, owner := range []string{"discovery", "navigate", "review", "rooms"} {
		t.Run(owner, func(t *testing.T) {
			text, err := execute("skills", "get", "nn-transcript", "--reference", owner)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(text, "--reference attention") {
				t.Fatalf("AD1_DISPATCH FAIL: %s lacks attention owner", owner)
			}
			t.Log("AD1_DISPATCH PASS: " + owner)
		})
	}
}
