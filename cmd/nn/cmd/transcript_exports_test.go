package cmd

import (
	"encoding/json"
	"strings"
	"testing"
)

func exportObject(t *testing.T, v any) map[string]any {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	var m map[string]any
	if e = json.Unmarshal(b, &m); e != nil {
		t.Fatal(e)
	}
	return m
}

func TestTranscriptExportTreeProjection(t *testing.T) {
	rows := []agent{{ID: "ROOT", Cost: 3, SubtreeCost: 10}, {ID: "child", ParentID: "ROOT", Cost: 7, SubtreeCost: 7, Result: "private"}}
	v, err := projectTranscriptTree(rows, "child", "id,cost,subtree_cost,evidence_scope")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(v)
	var got []map[string]any
	_ = json.Unmarshal(b, &got)
	if len(got) != 1 || len(got[0]) != 4 || got[0]["id"] != "child" || got[0]["cost"] != float64(7) || got[0]["subtree_cost"] != float64(7) || got[0]["evidence_scope"] != nil {
		t.Fatal("ASSERT_EXPORT_TREE: fail")
	}
	t.Log("ASSERT_EXPORT_TREE: pass")
}
func TestTranscriptExportTreeReject(t *testing.T) {
	rows := []agent{{ID: "ROOT"}}
	for _, q := range [][2]string{{"missing", "id"}, {"ROOT", "typo"}, {"ROOT", "id,,cost"}} {
		if _, err := projectTranscriptTree(rows, q[0], q[1]); err == nil {
			t.Fatal("ASSERT_EXPORT_TREE_REJECT: fail")
		}
	}
	t.Log("ASSERT_EXPORT_TREE_REJECT: pass")
}
func TestTranscriptExportShow(t *testing.T) {
	text := strings.Repeat("α<&😀\n", 20000)
	for _, raw := range []bool{false, true} {
		v, err := completeTranscriptShow("session.jsonl", "agent", raw, text)
		if err != nil {
			t.Fatal(err)
		}
		got := exportObject(t, v)
		p, err := buildTranscriptShowPage("session.jsonl", "agent", raw, text, 1, "")
		if err != nil {
			t.Fatal(err)
		}
		if got["text"] != text || got["snapshot"] != p.Snapshot || got["mode"] != p.Mode || got["all"] != true {
			t.Fatal("ASSERT_EXPORT_SHOW: fail")
		}
	}
	t.Log("ASSERT_EXPORT_SHOW: pass")
}
func TestTranscriptExportEvents(t *testing.T) {
	payload, _ := json.Marshal(map[string]any{"event_id": "one", "payload": strings.Repeat("😀", 20000)})
	p := ledgerPage{Snapshot: "retained", Events: []json.RawMessage{}}
	got := completeLedgerExport(p, []json.RawMessage{payload})
	m := exportObject(t, got)
	if m["all"] != true || got.Snapshot != "retained" || len(got.Events) != 1 || string(got.Events[0]) != string(payload) || got.NextPage != 0 {
		t.Fatal("ASSERT_EXPORT_EVENTS: fail")
	}
	t.Log("ASSERT_EXPORT_EVENTS: pass")
}
