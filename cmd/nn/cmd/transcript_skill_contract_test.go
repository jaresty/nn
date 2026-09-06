package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedTranscriptSkillShowPaginationContractMatchesCLI(t *testing.T) {
	const assertion = "ASSERT_EMBEDDED_TRANSCRIPT_SKILL_SHOW_PAGINATION_CONTRACT_MATCHES_CLI"
	cmd := newTranscriptShowCmd()
	for _, name := range []string{"raw", "json", "page", "snapshot"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("%s: show flag --%s is absent", assertion, name)
		}
	}
	root := filepath.Join("..", "..", "..")
	for _, path := range []string{
		filepath.Join(root, "skills", "nn-transcript", "SKILL.md"),
		filepath.Join(root, "skills", "nn-transcript", "references", "navigate.md"),
	} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: read %s: %v", assertion, path, err)
		}
		text := string(body)
		for _, required := range []string{"transcript show", "--json", "--snapshot", "every"} {
			if !strings.Contains(text, required) {
				t.Fatalf("%s: %s does not contain %q", assertion, path, required)
			}
		}
	}
}

// Validate actual command output rather than reflection on internal Go structs:
// accidental JSON-tag changes and numeric-to-string changes must fail this guard.
func TestEmbeddedTranscriptSkillJSONFieldContract(t *testing.T) {
	_, execute := setupNotebook(t)
	dir := t.TempDir()
	session := writePiFixture(t, dir)
	decode := func(args ...string) any {
		t.Helper()
		out, err := execute(append([]string{"transcript"}, args...)...)
		if err != nil {
			t.Fatal(err)
		}
		var value any
		if err := json.Unmarshal([]byte(out), &value); err != nil {
			t.Fatal(err)
		}
		return value
	}
	fields := func(value any, stringFields, numberFields, boolFields string) {
		t.Helper()
		obj, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("expected JSON object, got %T", value)
		}
		for _, name := range strings.Fields(stringFields) {
			if _, ok := obj[name].(string); !ok {
				t.Errorf("ASSERT_TRANSCRIPT_SKILL_JSON_FIELDS: %s must be a string, got %T", name, obj[name])
			}
		}
		for _, name := range strings.Fields(numberFields) {
			if _, ok := obj[name].(float64); !ok {
				t.Errorf("ASSERT_TRANSCRIPT_SKILL_JSON_FIELDS: %s must be numeric, got %T", name, obj[name])
			}
		}
		for _, name := range strings.Fields(boolFields) {
			if _, ok := obj[name].(bool); !ok {
				t.Errorf("ASSERT_TRANSCRIPT_SKILL_JSON_FIELDS: %s must be boolean, got %T", name, obj[name])
			}
		}
	}
	first := func(value any) any {
		t.Helper()
		rows, ok := value.([]any)
		if !ok || len(rows) == 0 {
			t.Fatalf("expected nonempty JSON array, got %T", value)
		}
		return rows[0]
	}
	lsRow := first(decode("ls", dir, "--json")).(map[string]any)
	fields(lsRow, "session path modified schema tree_preview cursor", "agent_count total_cost", "")
	summary, ok := lsRow["summary"].(map[string]any)
	if !ok {
		t.Fatal("ASSERT_TRANSCRIPT_SKILL_JSON_FIELDS: summary must be an object")
	}
	fields(summary, "topology_status", "distinct_agent_types omitted_type_count omitted_agent_count", "types_truncated")
	fields(summary["cost"], "status", "total_tokens input_tokens output_tokens cache_read_tokens cache_creation_tokens measured_agents unavailable_agents", "")
	fields(summary["topology"], "", "root_count edge_count max_depth max_children", "")
	fields(first(summary["agent_types"]), "type", "count", "")
	treeRow := first(decode("tree", session, "--json")).(map[string]any)
	fields(treeRow, "id parent_id type started ended status result cost_status subtree_cost_status",
		"cost subtree_cost input_tokens output_tokens cache_read_tokens cache_creation_tokens", "")
	fields(treeRow["evidence_scope"], "status timestamps cost subtree_cost", "terminal_record_count", "")
	page := decode("show", session, "ROOT", "--json")
	fields(page, "snapshot mode", "page pages next_page", "")
	fields(first(page.(map[string]any)["segments"]), "text", "segment segments", "")
	search := decode("search", "Agent", "--session", session, "--json")
	fields(search, "", "returned", "truncated")
	fields(first(search.(map[string]any)["matches"]), "session agent_id event_id timestamp role excerpt source_path", "", "")
}

func TestEmbeddedTranscriptSkillEvidenceBoundary(t *testing.T) {
	root := filepath.Join("..", "..", "..", "skills", "nn-transcript")
	for name, required := range map[string][]string{
		"SKILL.md":               {"--cursor", "tree_preview", "total_cost", "cost_status", "subtree_cost_status", "token counts, not currency", "exact topology requires", "summary.cost.status", "topology_status", "omitted_agent_count", "summary: null", "evidence_scope", "last_terminal_record", "retained_sidechain_history"},
		"references/navigate.md": {"token counts", "cost_status", "subtree_cost_status", "parent_id", "started", "ended", "evidence_scope", "terminal_record_count", "not task success"},
		"references/patterns.md": {"--cursor", "tree --json", "token counts", "summary.cost", "summary.topology", "types_truncated"},
	} {
		body, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		for _, text := range required {
			if !strings.Contains(string(body), text) {
				t.Errorf("ASSERT_TRANSCRIPT_SKILL_EVIDENCE_BOUNDARY: %s lacks %q", name, text)
			}
		}
		for _, forbidden := range []string{"$24.5k", "ls --before <oldest.modified>", "repeated agent-type, repeated depth"} {
			if strings.Contains(string(body), forbidden) {
				t.Errorf("ASSERT_TRANSCRIPT_SKILL_EVIDENCE_BOUNDARY: %s retains %q", name, forbidden)
			}
		}
	}
}

func TestEmbeddedTranscriptSkillSearchContractMatchesCLI(t *testing.T) {
	const assertion = "ASSERT_EMBEDDED_TRANSCRIPT_SKILL_SEARCH_CONTRACT_MATCHES_CLI"
	cmd := newTranscriptSearchCmd()
	for _, name := range []string{"session", "agent", "before", "raw", "json", "limit"} {
		if cmd.Flags().Lookup(name) == nil {
			t.Fatalf("%s: search flag --%s is absent", assertion, name)
		}
	}
	root := filepath.Join("..", "..", "..")
	paths := []string{
		filepath.Join(root, "skills", "nn-transcript", "SKILL.md"),
		filepath.Join(root, "skills", "nn-transcript", "references", "patterns.md"),
	}
	for _, path := range paths {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: read %s: %v", assertion, path, err)
		}
		text := string(body)
		if !strings.Contains(text, "nn transcript search") {
			t.Fatalf("%s: %s does not name the serving search command", assertion, path)
		}
	}
	core, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(core), "`nn grep") && !strings.Contains(string(core), "Never use `nn grep`") {
		t.Fatalf("%s: skill retains an nn grep instruction", assertion)
	}
}
