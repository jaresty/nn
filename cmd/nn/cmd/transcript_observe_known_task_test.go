package cmd

import (
	"strings"
	"testing"
)

// Execute the served known-task recipe; this tests publication and native
// capability, not whether a conversational model chooses the correct branch.
func TestObserveKnownTaskPublishedRecipe(t *testing.T) {
	for _, schema := range []string{"pi", "sdk-cli", "claude-code"} {
		t.Run(schema, func(t *testing.T) {
			_, execute := setupNotebook(t)
			path, id := attentionFixture(t, schema)
			body, err := execute("skills", "get", "nn-transcript", "--reference", "observe")
			if err != nil {
				t.Fatal(err)
			}
			template := ""
			for _, line := range strings.Split(body, "\n") {
				if value, ok := strings.CutPrefix(line, "Known task and exact agent(s): `"); ok {
					template = strings.TrimSuffix(value, "`")
				}
			}
			if template == "" {
				t.Fatal("KNOWN_TASK_RECIPE FAIL: missing combined recipe")
			}
			if strings.Index(body, template) > strings.Index(body, "\nnn transcript observe <session>\n") {
				t.Fatal("KNOWN_TASK_RECIPE FAIL: bare invocation takes precedence")
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
			for _, want := range []string{"## ROOT —", "Selected attention IDs: " + id, "match=1; no_match=0; indeterminate=0", "Inspect evidence: nn transcript attention inspect"} {
				if !strings.Contains(out, want) {
					t.Fatalf("KNOWN_TASK_RECIPE FAIL: missing %q", want)
				}
			}
			t.Log("KNOWN_TASK_RECIPE PASS: one published acquisition includes streams and classified attention")
		})
	}
}
