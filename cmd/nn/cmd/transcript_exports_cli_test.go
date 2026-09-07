package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestTranscriptExportCLI(t *testing.T) {
	_, execute := setupNotebook(t)
	session := writeSDKCLIFixture(t, t.TempDir())
	out, err := execute("transcript", "tree", session, "--json", "--agent", "aaa", "--fields", "id,cost,subtree_cost")
	if err != nil {
		t.Fatal(err)
	}
	var rows []map[string]any
	if err = json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["id"] != "aaa" || rows[0]["cost"] != float64(30) || rows[0]["subtree_cost"] != float64(75) || len(rows[0]) != 3 {
		t.Fatal("ASSERT_EXPORT_ROLLUP: fail")
	}
	t.Log("ASSERT_EXPORT_ROLLUP: pass")
	parent, _ := nativeToolResultFixture(t, true)
	for _, raw := range []bool{false, true} {
		args := []string{"transcript", "show", parent, "AAA"}
		if raw {
			args = append(args, "--raw")
		}
		plain, err := execute(args...)
		if err != nil {
			t.Fatal(err)
		}
		full, err := execute(append(args, "--all", "--json")...)
		if err != nil {
			t.Fatal(err)
		}
		var obj map[string]any
		if err = json.Unmarshal([]byte(full), &obj); err != nil {
			t.Fatal(err)
		}
		page, err := execute(append(args, "--json")...)
		if err != nil {
			t.Fatal(err)
		}
		var bounded transcriptShowPage
		_ = json.Unmarshal([]byte(page), &bounded)
		if obj["text"] != plain || obj["snapshot"] != bounded.Snapshot || obj["all"] != true {
			t.Fatal("ASSERT_EXPORT_SHOW_CLI: fail")
		}
	}
	t.Log("ASSERT_EXPORT_SHOW_CLI: pass")
	complete, p := ledgerAll(t, execute, parent, "AAA", "--payload")
	full, err := execute("transcript", "events", parent, "AAA", "--payload", "--all")
	if err != nil {
		t.Fatal(err)
	}
	var obj struct {
		All      bool             `json:"all"`
		Snapshot string           `json:"snapshot"`
		Events   []map[string]any `json:"events"`
	}
	if err = json.Unmarshal([]byte(full), &obj); err != nil {
		t.Fatal(err)
	}
	want, _ := json.Marshal(complete)
	got, _ := json.Marshal(obj.Events)
	if !obj.All || obj.Snapshot != p.Snapshot || string(want) != string(got) || len(full) <= 48000 {
		t.Fatal("ASSERT_EXPORT_EVENTS_CLI: fail")
	}
	t.Log("ASSERT_EXPORT_EVENTS_CLI: pass")
}

func TestTranscriptExportValidatesWholeTree(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pi.jsonl")
	writeTranscriptFile(t, path, `{"type":"session"}`+"\n"+
		`{"type":"message","id":"sa","agentId":"A","message":{"role":"assistant","content":[{"type":"toolCall","name":"Agent"}]}}`+"\n"+
		`{"type":"message","id":"sb","agentId":"B","message":{"role":"assistant","content":[{"type":"toolCall","name":"Agent"}]}}`+"\n"+
		`{"type":"custom","customType":"subagents:record","parentId":"sb","data":{"id":"A"}}`+"\n"+
		`{"type":"custom","customType":"subagents:record","parentId":"sa","data":{"id":"B"}}`+"\n")
	_, execute := setupNotebook(t)
	_, err := execute("transcript", "tree", path, "--json", "--strict", "--agent", "ROOT", "--fields", "id")
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("ASSERT_EXPORT_VALIDATION: fail — %v", err)
	}
	t.Log("ASSERT_EXPORT_VALIDATION: pass")
}

func TestTranscriptExportFlags(t *testing.T) {
	session := writePiFixture(t, t.TempDir())
	_, execute := setupNotebook(t)
	for _, args := range [][]string{
		{"tree", session, "--agent", "ROOT"}, {"tree", session, "--fields", "id"},
		{"tree", session, "--json", "--agent", ""}, {"tree", session, "--json", "--fields", ""},
		{"tree", session, "--json", "--agent", "missing"}, {"tree", session, "--json", "--fields", "bad"},
		{"show", session, "ROOT", "--all"}, {"show", session, "ROOT", "--all", "--json", "--page", "1"},
		{"show", session, "ROOT", "--all", "--json", "--snapshot", ""},
		{"events", session, "ROOT", "--all", "--page", "1"}, {"events", session, "ROOT", "--all", "--snapshot", ""},
	} {
		if _, err := execute(append([]string{"transcript"}, args...)...); err == nil {
			t.Fatalf("ASSERT_EXPORT_FLAGS: fail — %v", args)
		}
	}
	t.Log("ASSERT_EXPORT_FLAGS: pass")
}
