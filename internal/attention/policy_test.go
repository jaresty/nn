package attention

import (
	"strings"
	"testing"
)

func TestPolicyRuleControlsMatch(t *testing.T) {
	t.Log("procedure: TestPolicyRuleControlsMatch; assertion: P1_RULE")
	p, e := Builtin()
	if e != nil {
		t.Fatal(e)
	}
	m := Metrics{Commands: 40, Edits: 1, Available: true}
	r, e := p.Evaluate("thread", "implementation", m)
	if e != nil || r.Status != "match" {
		t.Fatalf("P1_RULE FAIL: %+v %v", r, e)
	}
	lower, e := parse(strings.Replace(builtin, "0.05", "0.01", 1))
	if e != nil {
		t.Fatal(e)
	}
	r, e = lower.Evaluate("thread", "implementation", m)
	if e != nil || r.Status != "no_match" {
		t.Fatalf("P1_RULE FAIL: threshold not rule-controlled: %+v %v", r, e)
	}
	opposite, e := parse(strings.Replace(builtin, "Ratio < Threshold", "Ratio >= Threshold", 1))
	if e != nil {
		t.Fatal(e)
	}
	r, e = opposite.Evaluate("thread", "implementation", m)
	if e != nil || r.Status != "no_match" {
		t.Fatalf("P1_RULE FAIL: operator not rule-controlled: %+v %v", r, e)
	}
	if lower.Digest == p.Digest {
		t.Fatal("P1_RULE FAIL: policy identity unchanged")
	}
	t.Log("P1_RULE PASS")
}

func TestPolicyOutcomes(t *testing.T) {
	t.Log("procedure: TestPolicyOutcomes; assertion: P3_OUTCOMES")
	p, e := Builtin()
	if e != nil {
		t.Fatal(e)
	}
	for _, tc := range []struct {
		name, task string
		m          Metrics
		want       string
	}{
		{"implementation", "implementation", Metrics{Commands: 40, Edits: 1, Available: true}, "match"},
		{"research", "research", Metrics{Commands: 40, Available: true}, "inapplicable"},
		{"verification", "verification", Metrics{Commands: 40, Available: true}, "inapplicable"},
		{"missing task", "", Metrics{Commands: 40, Available: true}, "inapplicable"},
		{"unknown", "implementation", Metrics{Commands: 40, Unknown: 1, Available: true}, "indeterminate"},
		{"unavailable", "implementation", Metrics{Commands: 40}, "indeterminate"},
		{"zero", "implementation", Metrics{Available: true}, "indeterminate"},
		{"exact ratio", "implementation", Metrics{Commands: 40, Edits: 2, Available: true}, "no_match"},
		{"exact minimum", "implementation", Metrics{Commands: 30, Available: true}, "match"},
		{"small sample", "implementation", Metrics{Commands: 29, Available: true}, "no_match"},
	} {
		r, e := p.Evaluate("t", tc.task, tc.m)
		if e != nil || r.Status != tc.want {
			t.Fatalf("P3_OUTCOMES FAIL: %s: %+v %v", tc.name, r, e)
		}
	}
	t.Log("P3_OUTCOMES PASS")
}

func TestBuiltinConditionChecks(t *testing.T) {
	p, err := Builtin()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		metrics Metrics
		status  string
		passed  []bool
	}{
		{"small-sample", Metrics{Commands: 15, Available: true}, "no_match", []bool{false, true}},
		{"high-ratio", Metrics{Commands: 40, Edits: 5, Available: true}, "no_match", []bool{true, false}},
		{"match", Metrics{Commands: 40, Available: true}, "match", []bool{true, true}},
		{"indeterminate", Metrics{Commands: 0, Available: true}, "indeterminate", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r, err := p.Evaluate("t", "implementation", tc.metrics)
			if err != nil || r.Status != tc.status || len(r.Checks) != len(tc.passed) {
				t.Fatalf("CONDITION_CHECKS FAIL: %+v %v", r, err)
			}
			for i, passed := range tc.passed {
				if r.Checks[i].Passed != passed {
					t.Fatalf("CONDITION_CHECKS FAIL: %+v", r.Checks)
				}
			}
		})
	}
	variant, err := parse(strings.Replace(builtin, "Ratio < Threshold", "Ratio >= Threshold", 1))
	if err != nil {
		t.Fatal(err)
	}
	r, err := variant.Evaluate("t", "implementation", Metrics{Commands: 40, Edits: 5, Available: true})
	if err != nil || len(r.Checks) != 0 {
		t.Fatalf("VARIANT_DIAGNOSTICS FAIL: %+v %v", r, err)
	}
}

func TestPolicyRejectsInvalid(t *testing.T) {
	t.Log("procedure: TestPolicyRejectsInvalid; assertion: P6_LIMITS")
	for _, src := range []string{
		builtin + "\nextra: true\n", builtin + "\n---\nid: second\n",
		strings.Replace(builtin, "minimum_commands: 30", "minimum_commands: 0", 1),
		strings.Replace(builtin, "minimum_commands: 30", "minimum_commands: .nan", 1),
		strings.Replace(builtin, "last_work_events: 100", "last_work_events: 10000", 1),
		strings.Replace(builtin, "Ratio < Threshold", "Edits / Commands < Threshold", 1),
		strings.Replace(builtin, "classification_complete(Thread)", "run_shell(Thread)", 1),
		strings.Replace(builtin, "command_count(Thread, Commands)", "attention(Thread, Commands)", 1),
		strings.Replace(builtin, "parameter(minimum_commands, Minimum)", "parameter(Unknown, Minimum)", 1),
		strings.Replace(builtin, "command_count(Thread, Commands)", "nonsense", 1),
	} {
		if _, e := parse(src); e == nil {
			t.Fatalf("P6_LIMITS FAIL: invalid definition accepted: %s", src)
		}
	}
	p, e := Builtin()
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.Evaluate("t", "implementation", Metrics{Commands: 2001}); e == nil {
		t.Fatal("P6_LIMITS FAIL: operation bound not enforced")
	}
	t.Log("P6_LIMITS PASS")
}
