package attention

import "testing"

func TestSignalApplicabilityAndCondition(t *testing.T) {
	p, err := Builtin()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, task, app, condition, outcome string
		m                                   Metrics
	}{
		{"unknown", "", "unknown", "match", "needs_context", Metrics{Commands: 40, Available: true}},
		{"implementation", "implementation", "applicable", "match", "triggered", Metrics{Commands: 40, Available: true}},
		{"negative", "implementation", "applicable", "no_match", "not_triggered", Metrics{Commands: 40, Edits: 4, Available: true}},
		{"other", "research", "not_applicable", "match", "not_applicable", Metrics{Commands: 40, Available: true}},
		{"unavailable", "", "unknown", "indeterminate", "insufficient_evidence", Metrics{}},
		{"error", "implementation", "applicable", "error", "error", Metrics{Commands: -1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := p.Observe("A", tc.task, "cohort_override", tc.m)
			if r.Applicability.Status != tc.app || r.Condition.Status != tc.condition || r.Outcome != tc.outcome {
				t.Fatalf("SIGNAL_SEPARATION FAIL: %+v", r)
			}
		})
	}
	// Existing callers keep historical missing-scope behavior.
	old, err := p.Evaluate("A", "", Metrics{Commands: 40, Available: true})
	if err != nil || old.Status != "inapplicable" {
		t.Fatal(old, err)
	}
	t.Log("SIGNAL_SEPARATION PASS")
}
