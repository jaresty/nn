package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranscriptDefaultRootsAreBoundedAndProviderSpecific(t *testing.T) {
	home := t.TempDir()
	codexHome := filepath.Join(home, "custom-codex")
	roots := transcriptDefaultRoots(home, func(key string) string {
		if key == "CODEX_HOME" {
			return codexHome
		}
		return ""
	})
	want := []transcriptRoot{
		{Source: "claude", Path: filepath.Join(home, ".claude", "projects")},
		{Source: "codex", Path: filepath.Join(codexHome, "sessions")},
		{Source: "codex", Path: filepath.Join(codexHome, "archived_sessions")},
		{Source: "pi", Path: filepath.Join(home, ".pi", "agent", "sessions")},
	}
	if len(roots) != len(want) {
		t.Fatalf("ASSERT_TRANSCRIPT_ROOTS_ARE_BOUNDED: got=%v want=%v", roots, want)
	}
	for i := range want {
		if roots[i] != want[i] {
			t.Fatalf("ASSERT_TRANSCRIPT_ROOTS_USE_VERIFIED_DEFAULTS: root[%d]=%+v want=%+v", i, roots[i], want[i])
		}
	}
}

func TestResolveTranscriptSessionExplicitPathWins(t *testing.T) {
	home := t.TempDir()
	explicit := filepath.Join(t.TempDir(), "same.jsonl")
	writeTranscriptFile(t, explicit, `{"type":"session","version":3,"id":"explicit"}`+"\n")
	writeTranscriptFile(t, filepath.Join(home, ".pi", "agent", "sessions", "project", "same.jsonl"), `{"type":"session","version":3,"id":"discovered"}`+"\n")

	got, err := resolveTranscriptSession(explicit, home, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	if got != explicit {
		t.Fatalf("ASSERT_TRANSCRIPT_EXPLICIT_PATH_HAS_PRIORITY: got=%q want=%q", got, explicit)
	}
}

func TestResolveTranscriptSessionUniqueIDUsesExactDiscoveredPath(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".pi", "agent", "sessions", "project", "unique.jsonl")
	writeTranscriptFile(t, path, `{"type":"session","version":3,"id":"unique"}`+"\n")

	got, err := resolveTranscriptSession("unique", home, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	canonical, _ := filepath.EvalSymlinks(path)
	canonical, _ = filepath.Abs(canonical)
	if got != canonical {
		t.Fatalf("ASSERT_TRANSCRIPT_UNIQUE_ID_RETURNS_INVENTORY_PATH: got=%q want=%q", got, canonical)
	}
}

func TestResolveTranscriptSessionAmbiguousIDFailsWithStableCandidates(t *testing.T) {
	home := t.TempDir()
	claude := filepath.Join(home, ".claude", "projects", "a", "duplicate.jsonl")
	pi := filepath.Join(home, ".pi", "agent", "sessions", "b", "duplicate.jsonl")
	writeTranscriptFile(t, claude, `{"type":"user","uuid":"u","message":{"role":"user","content":"x"}}`+"\n")
	writeTranscriptFile(t, pi, `{"type":"session","version":3,"id":"duplicate"}`+"\n")

	_, err := resolveTranscriptSession("duplicate", home, func(string) string { return "" })
	if err == nil {
		t.Fatal("ASSERT_TRANSCRIPT_AMBIGUOUS_ID_IS_REJECTED: expected error")
	}
	message := err.Error()
	if !strings.Contains(message, "ambiguous") || !strings.Contains(message, claude) || !strings.Contains(message, pi) {
		t.Fatalf("ASSERT_TRANSCRIPT_AMBIGUITY_LISTS_CANDIDATES: %v", err)
	}
	if strings.Index(message, claude) > strings.Index(message, pi) {
		t.Fatalf("ASSERT_TRANSCRIPT_AMBIGUITY_IS_STABLE: %v", err)
	}
}

func TestResolveTranscriptSessionUsesCodexMetadataID(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".codex", "sessions", "2026", "09", "10", "rollout-2026-09-10T00-00-00-codex-file.jsonl")
	writeTranscriptFile(t, path, `{"timestamp":"2026-09-10T00:00:00Z","type":"session_meta","payload":{"session_id":"codex-session-id","id":"codex-session-id","cwd":"."}}`+"\n")

	got, err := resolveTranscriptSession("codex-session-id", home, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	canonical, _ := filepath.EvalSymlinks(path)
	canonical, _ = filepath.Abs(canonical)
	if got != canonical {
		t.Fatalf("ASSERT_TRANSCRIPT_CODEX_METADATA_ID_RESOLVES: got=%q want=%q", got, canonical)
	}
	if schema := classifyTranscript(path); schema != schemaCodex {
		t.Fatalf("ASSERT_TRANSCRIPT_CODEX_SCHEMA_IS_RECOGNIZED: got=%q", schema)
	}
}

func TestCodexTreeAndEventsUseNormalizedRootStream(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rollout-codex.jsonl")
	writeTranscriptFile(t, path,
		`{"timestamp":"2026-09-10T00:00:00Z","type":"session_meta","payload":{"session_id":"codex","id":"codex","cwd":"."}}`+"\n"+
			`{"timestamp":"2026-09-10T00:00:01Z","type":"event_msg","payload":{"type":"user_message","message":"Hello from Codex","kind":"plain"}}`+"\n"+
			`{"timestamp":"2026-09-10T00:00:02Z","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"Codex reply"}]}}`+"\n")
	_, execute := setupNotebook(t)
	tree, err := execute("transcript", "tree", path)
	if err != nil || !strings.Contains(tree, "ROOT") {
		t.Fatalf("ASSERT_TRANSCRIPT_CODEX_TREE_HAS_ROOT: err=%v out=%s", err, tree)
	}
	events, err := execute("transcript", "events", path, "ROOT", "--last", "10", "--format", "text")
	if err != nil || !strings.Contains(events, "Hello from Codex") || !strings.Contains(events, "Codex reply") {
		t.Fatalf("ASSERT_TRANSCRIPT_CODEX_EVENTS_ARE_NORMALIZED: err=%v out=%s", err, events)
	}
	shown, err := execute("transcript", "show", path, "ROOT")
	if err != nil || !strings.Contains(shown, "Hello from Codex") || !strings.Contains(shown, "Codex reply") {
		t.Fatalf("ASSERT_TRANSCRIPT_CODEX_SHOW_IS_NORMALIZED: err=%v out=%s", err, shown)
	}
	searched, err := execute("transcript", "search", "Codex reply", path, "--json")
	if err != nil || !strings.Contains(searched, "Codex reply") {
		t.Fatalf("ASSERT_TRANSCRIPT_CODEX_SEARCH_IS_NORMALIZED: err=%v out=%s", err, searched)
	}
}

func TestTranscriptCommandsUseDefaultRootsAndSessionIDs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".pi", "agent", "sessions", "project", "command-id.jsonl")
	writeTranscriptFile(t, path,
		`{"type":"session","version":3,"id":"command-id","cwd":"/workspace"}`+"\n"+
			`{"type":"message","id":"m1","parentId":null,"timestamp":"2026-09-10T00:00:00Z","message":{"role":"user","content":"DEFAULT_ROOT_NEEDLE"}}`+"\n")
	_, execute := setupNotebook(t)

	listed, err := execute("transcript", "ls", "--json")
	if err != nil || !strings.Contains(listed, path) {
		t.Fatalf("ASSERT_TRANSCRIPT_LS_DEFAULTS_TO_PROVIDER_ROOTS: err=%v out=%s", err, listed)
	}
	scanned, err := execute("transcript", "scan")
	if err != nil || !strings.Contains(scanned, "pi: 1") {
		t.Fatalf("ASSERT_TRANSCRIPT_SCAN_DEFAULTS_TO_PROVIDER_ROOTS: err=%v out=%s", err, scanned)
	}
	searched, err := execute("transcript", "search", "DEFAULT_ROOT_NEEDLE", "--json")
	if err != nil || !strings.Contains(searched, "DEFAULT_ROOT_NEEDLE") {
		t.Fatalf("ASSERT_TRANSCRIPT_SEARCH_DEFAULTS_TO_PROVIDER_ROOTS: err=%v out=%s", err, searched)
	}
	tree, err := execute("transcript", "tree", "command-id")
	if err != nil || !strings.Contains(tree, "ROOT") {
		t.Fatalf("ASSERT_TRANSCRIPT_COMMAND_RESOLVES_SESSION_ID: err=%v out=%s", err, tree)
	}
}

func TestTranscriptAvailableDefaultRootsSkipsMissing(t *testing.T) {
	home := t.TempDir()
	pi := filepath.Join(home, ".pi", "agent", "sessions")
	if err := os.MkdirAll(pi, 0o755); err != nil {
		t.Fatal(err)
	}
	roots, unavailable, err := transcriptAvailableDefaultRoots(home, func(string) string { return "" })
	if err != nil {
		t.Fatal(err)
	}
	canonicalPi, _ := filepath.EvalSymlinks(pi)
	canonicalPi, _ = filepath.Abs(canonicalPi)
	if len(roots) != 1 || roots[0] != canonicalPi {
		t.Fatalf("ASSERT_TRANSCRIPT_DISCOVERY_USES_AVAILABLE_ROOTS: roots=%v", roots)
	}
	if len(unavailable) != 3 {
		t.Fatalf("ASSERT_TRANSCRIPT_DISCOVERY_REPORTS_UNAVAILABLE_ROOTS: unavailable=%v", unavailable)
	}
}
