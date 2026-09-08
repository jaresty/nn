package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranscriptToolSummaryCLI(t *testing.T) {
	session, _ := nativeToolResultFixture(t, true)
	_, execute := setupNotebook(t)
	out, err := execute("transcript", "events", session, "AAA", "--summary", "tools", "--group-by", "tool")
	if err != nil {
		t.Fatalf("ASSERT_TOOL_CLI: fail — %v", err)
	}
	var v map[string]any
	if err = json.Unmarshal([]byte(out), &v); err != nil {
		t.Fatal(err)
	}
	if v["version"] != "nn.transcript.tool-summary/v1" || v["stats"].(map[string]any)["results"] != float64(1) || strings.Contains(out, "FOREIGN_NATIVE_RESULT") {
		t.Fatal("ASSERT_TOOL_CLI: fail")
	}
	for _, flags := range [][]string{{"--summary", "tools", "--bucket-size", "0"}, {"--summary", "usage", "--limit", "5"}, {"--group-by", "tool"}, {"--summary", "tools", "--payload"}, {"--summary", "tools", "--page", "1"}} {
		if _, err := execute(append([]string{"transcript", "events", session, "AAA"}, flags...)...); err == nil {
			t.Fatalf("ASSERT_TOOL_CLI: fail — %v", flags)
		}
	}
	t.Log("ASSERT_TOOL_CLI: pass")
}
func TestTranscriptToolSummarySkill(t *testing.T) {
	for _, path := range []string{"references/summaries.md"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "..", "skills", "nn-transcript", path))
		if err != nil {
			t.Fatal(err)
		}
		for _, term := range []string{"--summary tools", "--group-by tool"} {
			if !strings.Contains(string(b), term) {
				t.Fatalf("ASSERT_TOOL_SKILL: fail — %s missing %s", path, term)
			}
		}
	}
	t.Log("ASSERT_TOOL_SKILL: pass")
}
