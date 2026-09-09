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
		{"AD1_SCOPE", "interaction", "For bare attention, the current surface scopes discovery before any background selected target."},
		{"AD2_PERMISSION", "interaction", "Target resolution is not acquisition permission."},
		{"AD3_APPROVAL", "attention", "Acceptance authorizes the declared selection and evaluation without per-room reconfirmation."},
		{"AD4_ATTEMPTS", "attention", "Count an attempted room when its assignment or work acquisition starts, including failures and unknowns."},
		{"AD5_OUTCOMES", "attention", "Task scope not established is distinct from known outside policy scope."},
		{"AD6_REPLACEMENT", "attention", "Do not silently replace an unavailable or unclassifiable candidate."},
		{"AD7_WIDENING", "attention", "An empty filtered population stays empty; do not clear the filter or widen discovery automatically."},
		{"AD8_BACK", "interaction", "Back never refunds consumed attention attempts or output allowance."},
		{"AD9_AUTOMATION", "attention", "Without standing approval, entering a view does not start an attention check."},
		{"AD10_EVALUATOR", "attention", "This discovery recipe does not change the native evaluator or its policy."},
		{"AD11_IMPERATIVE", "interaction", "A clear imperative authorizes its ordinary bounded read-only operation."},
		{"AD12_RECOVERY", "attention", "An explicit recovery-check request authorizes one bounded fresh evaluation."},
		{"AD13_NO_CONFIRM", "attention", "Do not ask “Run recovery check?” after the user has already requested that check."},
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
func TestAttentionStandingApproval(t *testing.T) {
	_, execute := setupNotebook(t)
	for _, clause := range []string{
		"Standing approval authorizes eligible Open and explicit Refresh checks without per-check reconfirmation.",
		"One navigation action starts at most one attention pass.",
		"Back, Forward restoration, re-rendering, pagination, and tool completion do not trigger a check.",
		"Every pass consumes the standing allowance; no navigation action resets cumulative consumption.",
		"Opt-out, End, a new conversation, or lost authorization state disables standing attention.",
		"Exhaustion pauses checks; it does not renew permission.",
		"Standing attention does not authorize deeper inspection, capture, or intervention.",
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
			if !strings.Contains(strings.ToLower(text), "standing attention") {
				t.Fatalf("STANDING_DISPATCH FAIL: %s", owner)
			}
		})
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
		if task != "implementation" && p.Rooms[0].Result.Status != "inapplicable" {
			t.Fatal("native missing/other scope semantics changed")
		}
	}
	t.Log("NATIVE_RECIPE_PASS: one metadata page, one context page, exact selected room; native scope semantics unchanged")
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
