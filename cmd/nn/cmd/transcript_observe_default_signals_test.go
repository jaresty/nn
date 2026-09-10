package cmd

import (
	"strings"
	"testing"
)

func TestObserveDefaultPublishedSignals(t *testing.T) {
	for _, schema := range []string{"pi", "sdk-cli", "claude-code"} {
		t.Run(schema, func(t *testing.T) {
			_, execute := setupNotebook(t)
			path, _ := attentionFixture(t, schema)
			body, err := execute("skills", "get", "nn-transcript", "--reference", "observe")
			if err != nil {
				t.Fatal(err)
			}
			template := ""
			for _, line := range strings.Split(body, "\n") {
				if line == "nn transcript observe <session-id>" {
					template = line
				}
			}
			if template == "" {
				t.Fatal("DEFAULT_SIGNALS FAIL: missing default recipe")
			}
			args := strings.Fields(template)[1:]
			args[len(args)-1] = path
			out, err := execute(args...)
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range []string{"## ROOT —", "Signal: low-edit-ratio", "Outcome: needs_context", "Condition: match", "Task context:", "Inspect evidence: nn transcript attention inspect"} {
				if !strings.Contains(out, s) {
					t.Fatalf("DEFAULT_SIGNALS FAIL: missing %q", s)
				}
			}
			if strings.Contains(out, "not evaluated — task scope not established") {
				t.Fatal("DEFAULT_SIGNALS FAIL: activation switch restored")
			}
			t.Log("DEFAULT_SIGNALS PASS: one unclassified observation returns measurements and evidence")
		})
	}
}
