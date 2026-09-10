package cmd

import (
	"strings"
	"testing"
)

// Execute the served exact-target recipe without classification flags. This tests
// publication and native capability, not conversational model compliance.
func TestObserveExactTargetPublishedRecipe(t *testing.T) {
	for _, schema := range []string{"pi", "sdk-cli", "claude-code"} {
		t.Run(schema, func(t *testing.T) {
			_, execute := setupNotebook(t)
			path, id := attentionFixture(t, schema)
			body, err := execute("skills", "get", "nn-transcript", "--reference", "observe")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(body, "Signal: <signal_id> · policy v<version> · metric v<metric_version>") {
				t.Fatal("SIGNAL_LABEL FAIL: missing explicit signal identity recipe")
			}
			for _, flag := range []string{"--task", "--agent-task"} {
				if strings.Contains(body, flag) {
					t.Fatalf("EXACT_TARGET_RECIPE FAIL: normal observation advertises %s", flag)
				}
			}
			template := ""
			for _, line := range strings.Split(body, "\n") {
				if value, ok := strings.CutPrefix(line, "Exact target: `"); ok {
					template = strings.TrimSuffix(value, "`")
				}
			}
			if template == "" {
				t.Fatal("EXACT_TARGET_RECIPE FAIL: missing combined recipe")
			}
			args := strings.Fields(template)[1:]
			for i, arg := range args {
				switch arg {
				case "<session>":
					args[i] = path
				case "<id>":
					args[i] = id
				}
			}
			// Exactly one acquisition command from the published template; no attention
			// reference load, classification lookup, or standalone attention invocation.
			out, err := execute(args...)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range []string{"## ROOT —", "Selected attention IDs: " + id, "triggered=0; not_triggered=0; needs_context=1", "Condition: match", "Inspect evidence: nn transcript attention inspect"} {
				if !strings.Contains(out, want) {
					t.Fatalf("EXACT_TARGET_RECIPE FAIL: missing %q", want)
				}
			}
			t.Log("EXACT_TARGET_RECIPE PASS: one published acquisition includes streams and automatic attention without classification flags")
		})
	}
}
