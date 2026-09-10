package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/attention"
)

func TestAttentionHypotheticalTable(t *testing.T) {
	tests := []struct{ name, app, condition, want string }{
		{"match", "unknown", "match", "If this is an implementation task, this signal would trigger."},
		{"no-match", "unknown", "no_match", "If this is an implementation task, this signal would not trigger."},
		{"indeterminate", "unknown", "indeterminate", "If this is an implementation task, available evidence is insufficient."},
		{"applicable", "applicable", "match", ""},
		{"not-applicable", "not_applicable", "match", ""},
		{"error", "unknown", "error", ""},
		{"old-receipt", "unknown", "match", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scope := "implementation"
			if tc.name == "old-receipt" {
				scope = ""
			}
			s := attentionSignal{Scope: scope, SignalResult: attention.SignalResult{
				Applicability: attention.Applicability{Status: tc.app},
				Condition:     attention.Result{Status: tc.condition},
			}}
			if got := attentionHypothetical(s); got != tc.want {
				t.Fatalf("HYPOTHETICAL_TABLE FAIL: %q != %q", got, tc.want)
			}
		})
	}
}

func TestAttentionCollapsedSignalExplainsHypothetical(t *testing.T) {
	signal := attentionSignal{
		ID: "low-edit-ratio", Version: 1, Digest: "digest", Scope: "implementation",
		SignalResult: attention.SignalResult{
			Applicability: attention.Applicability{Status: "unknown", Source: "not_established", Reason: "Task classification not established; condition measured independently"},
			Condition: attention.Result{Status: "no_match", Reason: "Rule did not match", Checks: []attention.ConditionCheck{
				{Name: "minimum_commands", Actual: 15, Operator: ">=", Threshold: 30},
				{Name: "maximum_edit_ratio", Actual: 0, Operator: "<", Threshold: 0.05, Passed: true},
			}},
			Outcome: "needs_context",
		},
	}
	p := attentionPage{Version: 2, MetricVersion: 2, Rooms: []attentionRoom{{ID: "A", Signals: []attentionSignal{signal}}}}
	var out bytes.Buffer
	if err := renderAttentionSignals(&out, p); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"Signal: low-edit-ratio v1 · scope: implementation",
		"Applicability: unknown",
		"Condition: no_match",
		"Check: minimum_commands · 15 >= 30 — failed",
		"Check: maximum_edit_ratio · 0 < 0.05 — passed",
		"Hypothetical: If this is an implementation task, this signal would not trigger because only 15 of the required 30 commands were observed.",
		"Outcome: needs_context",
	} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("COLLAPSED_HYPOTHETICAL FAIL: missing %q\n%s", want, out.String())
		}
	}
}

func TestAttentionHypotheticalExplainsFailedCheck(t *testing.T) {
	for _, tc := range []struct {
		name  string
		check attention.ConditionCheck
		want  string
	}{
		{"commands", attention.ConditionCheck{Name: "minimum_commands", Actual: 15, Operator: ">=", Threshold: 30}, "because only 15 of the required 30 commands were observed."},
		{"ratio", attention.ConditionCheck{Name: "maximum_edit_ratio", Actual: 0.125, Operator: "<", Threshold: 0.05}, "because the edit ratio was 0.125; triggering requires less than 0.05."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := attentionSignal{Scope: "implementation", SignalResult: attention.SignalResult{Applicability: attention.Applicability{Status: "unknown"}, Condition: attention.Result{Status: "no_match", Checks: []attention.ConditionCheck{tc.check}}}}
			if got := attentionHypothetical(s); !strings.Contains(got, tc.want) {
				t.Fatalf("FAILED_CHECK_EXPLANATION FAIL: %q", got)
			}
		})
	}
}

func TestObserveCoverageExplainsHypothetical(t *testing.T) {
	signal := attentionSignal{ID: "low-edit-ratio", Scope: "implementation", SignalResult: attention.SignalResult{
		Applicability: attention.Applicability{Status: "unknown"},
		Condition:     attention.Result{Status: "match"}, Outcome: "needs_context",
	}}
	state := observeLiveState{At: time.Now(), Agents: []observeLiveAgent{{ID: "A", State: "evaluated", Room: &attentionRoom{Signals: []attentionSignal{signal}}}}}
	text, err := renderObserveCoverage(state, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"scope: implementation", "Hypothetical: If this is an implementation task, this signal would trigger."} {
		if !strings.Contains(text, want) {
			t.Fatalf("COVERAGE_HYPOTHETICAL FAIL: missing %q\n%s", want, text)
		}
	}
}

func TestAttentionApplicableHasNoHypothetical(t *testing.T) {
	s := attentionSignal{Scope: "implementation", SignalResult: attention.SignalResult{
		Applicability: attention.Applicability{Status: "applicable"},
		Condition:     attention.Result{Status: "match"},
	}}
	if attentionHypothetical(s) != "" {
		t.Fatal("DIRECT_RESULT_HYPOTHETICAL FAIL")
	}
}
