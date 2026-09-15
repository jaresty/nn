package skills

import (
	"strings"
	"testing"
)

func TestCaptureDisciplineIsToolNeutralAndNonDisplacing(t *testing.T) {
	body, err := FS.ReadFile("nn-capture-discipline/SKILL.md")
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if strings.Contains(strings.ToLower(text), "lsp-trace") {
		t.Fatalf("ASSERT_CAPTURE_SKILL_TOOL_NEUTRAL: product-specific tool name found")
	}
	if !strings.Contains(text, "This workflow governs only the file-reading and external-service actions named above") {
		t.Fatalf("ASSERT_CAPTURE_SKILL_STRUCTURAL_SCOPE: missing structural scope boundary")
	}
	if !strings.Contains(text, "Do not substitute an nn retrieval command for another available tool that directly observes the requested relation") {
		t.Fatalf("ASSERT_CAPTURE_SKILL_NO_SEMANTIC_DISPLACEMENT: missing tool-neutral routing boundary")
	}
}
