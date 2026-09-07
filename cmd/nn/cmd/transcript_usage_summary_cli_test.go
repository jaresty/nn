package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranscriptUsageSummaryCLI(t *testing.T) {
	session, _ := nativeToolResultFixture(t, true)
	_, execute := setupNotebook(t)
	out, err := execute("transcript", "events", session, "AAA", "--summary", "usage", "--bucket-size", "1")
	if err != nil {
		t.Fatalf("ASSERT_USAGE_CLI: fail — %v", err)
	}
	var result map[string]any
	if err = json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	stats := result["stats"].(map[string]any)
	if result["version"] != "nn.transcript.usage-summary/v1" || stats["records"] != float64(1) || stats["known_total_tokens"] != float64(7) || stats["status"] != "partial" || result["events"] != nil || strings.Contains(out, "OWNED_NATIVE_RESULT") {
		t.Fatal("ASSERT_USAGE_CLI: fail")
	}
	ledger, err := execute("transcript", "events", session, "AAA", "--select", "identity,usage", "--all")
	if err != nil {
		t.Fatal(err)
	}
	var l map[string]any
	_ = json.Unmarshal([]byte(ledger), &l)
	if l["snapshot"] != result["ledger_snapshot"] {
		t.Fatal("ASSERT_USAGE_CLI: fail — ledger provenance mismatch")
	}
	snapshot := result["snapshot"].(string)
	if _, err := execute("transcript", "events", session, "AAA", "--summary", "usage", "--bucket-size", "1", "--snapshot", snapshot); err != nil {
		t.Fatal(err)
	}
	if _, err := execute("transcript", "events", session, "AAA", "--summary", "usage", "--bucket-size", "2", "--snapshot", snapshot); err == nil {
		t.Fatal("ASSERT_USAGE_CLI: fail — bucket snapshot mismatch accepted")
	}
	t.Log("ASSERT_USAGE_CLI: pass")
}
func TestTranscriptUsageSummarySkill(t *testing.T) {
	for _, path := range []string{"SKILL.md", "references/navigate.md", "references/patterns.md"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "..", "skills", "nn-transcript", path))
		if err != nil {
			t.Fatal(err)
		}
		for _, term := range []string{"--summary usage", "--bucket-size"} {
			if !strings.Contains(string(b), term) {
				t.Fatalf("ASSERT_USAGE_SKILL: fail — %s missing %s", path, term)
			}
		}
	}
	t.Log("ASSERT_USAGE_SKILL: pass")
}

func TestTranscriptUsageSummaryFlags(t *testing.T) {
	session := writePiFixture(t, t.TempDir())
	_, execute := setupNotebook(t)
	for _, flags := range [][]string{{"--summary", "bad"}, {"--summary", ""}, {"--bucket-size", "0"}, {"--summary", "usage", "--bucket-size", "-1"}, {"--summary", "usage", "--page", "1"}, {"--summary", "usage", "--all"}, {"--summary", "usage", "--payload=false"}, {"--summary", "usage", "--select", "identity,usage"}, {"--summary", "usage", "--event", ""}} {
		if _, err := execute(append([]string{"transcript", "events", session, "ROOT"}, flags...)...); err == nil {
			t.Fatalf("ASSERT_USAGE_FLAGS: fail — %v", flags)
		}
	}
	t.Log("ASSERT_USAGE_FLAGS: pass")
}
