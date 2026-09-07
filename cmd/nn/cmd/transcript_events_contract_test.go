package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func ledgerAll(t *testing.T, execute func(...string) (string, error), session, id string, flags ...string) ([]map[string]any, ledgerPage) {
	t.Helper()
	var result []map[string]any
	var p ledgerPage
	page := 1
	snapshot := ""
	partials := map[string]string{}
	next := map[string]int{}
	for {
		args := append([]string{"transcript", "events", session, id, "--json", "--page", fmtInt(page)}, flags...)
		if page > 1 {
			args = append(args, "--snapshot", snapshot)
		}
		out, err := execute(args...)
		if err != nil {
			t.Fatal(err)
		}
		if len(out) > 48000 {
			t.Fatal("ASSERT_LEDGER_PAGE_BOUND")
		}
		if err := json.Unmarshal([]byte(out), &p); err != nil {
			t.Fatal(err)
		}
		if page == 1 {
			snapshot = p.Snapshot
		} else if p.Snapshot != snapshot {
			t.Fatal("ASSERT_LEDGER_SNAPSHOT")
		}
		for _, raw := range p.Events {
			var e map[string]any
			if err := json.Unmarshal(raw, &e); err != nil {
				t.Fatal(err)
			}
			if count, ok := e["segments"].(float64); ok {
				key := e["event_id"].(string)
				n := int(e["segment"].(float64))
				if n != next[key]+1 {
					t.Fatal("ASSERT_LEDGER_SEGMENTS")
				}
				next[key] = n
				partials[key] += e["text"].(string)
				if n < int(count) {
					continue
				}
				// Decode into a fresh map: fragment transport keys are not event fields.
				e = nil
				if err := json.Unmarshal([]byte(partials[key]), &e); err != nil {
					t.Fatal(err)
				}
				delete(partials, key)
			}
			result = append(result, e)
		}
		if p.NextPage == 0 {
			break
		}
		page = p.NextPage
	}
	if len(partials) > 0 {
		t.Fatal("ASSERT_LEDGER_SEGMENTS")
	}
	return result, p
}
func fmtInt(n int) string { b, _ := json.Marshal(n); return string(b) }

func TestTranscriptEventsProjection(t *testing.T) {
	dir := t.TempDir()
	session := filepath.Join(dir, "pi.jsonl")
	writeTranscriptFile(t, session, `{"type":"session"}`+"\n"+`{"type":"message","id":"duplicate","message":{"role":"assistant","model":"model-a","timestamp":1234,"usage":{"input":10,"output":2,"cacheRead":100,"cacheWrite":0},"content":[{"type":"toolCall","id":"a","name":"bash","arguments":{"command":"SECRET"}},{"type":"toolCall","id":"b","name":"read","arguments":{}}]}}`+"\n"+`{"type":"toolResult","id":"duplicate","message":{"role":"toolResult","toolCallId":"a","toolName":"bash","isError":false,"usage":{"input":999},"content":[{"type":"text","text":"😀é"}]}}`+"\n"+`{"type":"message","message":{"role":"assistant","usage":{"input":0},"content":"end"}}`+"\n")
	_, execute := setupNotebook(t)
	events, p := ledgerAll(t, execute, session, "ROOT")
	if p.DetailStatus != "available" || len(events) != 6 {
		t.Fatalf("ASSERT_LEDGER_EXTRACTION: %+v", events)
	}
	t.Run("Identity", func(t *testing.T) {
		ids := map[string]bool{}
		for i, e := range events {
			id := e["event_id"].(string)
			if ids[id] || e["ordinal"] != float64(i+1) {
				t.Fatal("ASSERT_LEDGER_IDENTITY: fail")
			}
			ids[id] = true
		}
		t.Log("ASSERT_LEDGER_IDENTITY: pass")
	})
	t.Run("UsageOnce", func(t *testing.T) {
		count := 0
		var known float64
		for _, e := range events {
			if u, ok := e["usage"].(map[string]any); ok {
				count++
				known += u["known_total_tokens"].(float64)
				if e["kind"] != "message" {
					t.Fatal("ASSERT_LEDGER_USAGE_ONCE: fail")
				}
			}
		}
		if count != 2 || known != 112 {
			t.Fatal("ASSERT_LEDGER_USAGE_ONCE: fail")
		}
		t.Log("ASSERT_LEDGER_USAGE_ONCE: pass")
	})
	t.Run("Counters", func(t *testing.T) {
		u := events[0]["usage"].(map[string]any)
		if u["context_tokens"] != float64(110) || u["total_tokens"] != float64(112) || u["status"] != "complete" {
			t.Fatal("ASSERT_LEDGER_COUNTERS: fail")
		}
		t.Log("ASSERT_LEDGER_COUNTERS: pass")
	})
	t.Run("UnknownZero", func(t *testing.T) {
		u := events[5]["usage"].(map[string]any)
		if u["input_tokens"] != float64(0) || u["output_tokens"] != nil || u["status"] != "partial" || u["total_tokens"] != nil {
			t.Fatal("ASSERT_LEDGER_UNKNOWN_ZERO: fail")
		}
		t.Log("ASSERT_LEDGER_UNKNOWN_ZERO: pass")
	})
	t.Run("Join", func(t *testing.T) {
		call := events[1]["tools"].(map[string]any)
		res := events[4]["tools"].(map[string]any)
		if call["match_status"] != "matched" || call["matched_event_id"] != events[4]["event_id"] || res["matched_event_id"] != events[1]["event_id"] || res["is_error"] != false || events[2]["tools"].(map[string]any)["match_status"] != "missing" {
			t.Fatal("ASSERT_LEDGER_JOIN: fail")
		}
		t.Log("ASSERT_LEDGER_JOIN: pass")
	})
	t.Run("Size", func(t *testing.T) {
		size := events[4]["tools"].(map[string]any)["result_size"].(map[string]any)
		if size["text_bytes"] != float64(6) || size["text_characters"] != float64(2) {
			t.Fatal("ASSERT_LEDGER_SIZE: fail")
		}
		t.Log("ASSERT_LEDGER_SIZE: pass")
	})
	t.Run("Select", func(t *testing.T) {
		narrow, _ := ledgerAll(t, execute, session, "ROOT", "--select", "usage")
		for i, e := range narrow {
			if e["event_id"] != events[i]["event_id"] || e["message"] != nil || e["tools"] != nil {
				t.Fatal("ASSERT_LEDGER_SELECT: fail")
			}
		}
		t.Log("ASSERT_LEDGER_SELECT: pass")
	})
	t.Run("NoPayload", func(t *testing.T) {
		for _, e := range events {
			if _, ok := e["payload"]; ok {
				t.Fatal("ASSERT_LEDGER_NO_PAYLOAD: fail")
			}
		}
		out, _ := execute("transcript", "events", session, "ROOT")
		if strings.Contains(out, "SECRET") {
			t.Fatal("ASSERT_LEDGER_NO_PAYLOAD: fail")
		}
		t.Log("ASSERT_LEDGER_NO_PAYLOAD: pass")
	})
}

func TestTranscriptEventsOwnershipAndSchemas(t *testing.T) {
	parent, side := nativeToolResultFixture(t, true)
	_, execute := setupNotebook(t)
	direct, _ := ledgerAll(t, execute, side, "AAA")
	resolved, _ := ledgerAll(t, execute, parent, "AAA")
	a, _ := json.Marshal(direct)
	b, _ := json.Marshal(resolved)
	if string(a) != string(b) {
		t.Fatal("ASSERT_LEDGER_SHARED_SELECTION")
	}
	if len(direct) != 3 {
		t.Fatal("ASSERT_LEDGER_OWNERSHIP")
	}
	for _, e := range direct {
		source := e["source"].(map[string]any)
		if source["record_ordinal"].(float64) > 2 {
			t.Fatal("ASSERT_LEDGER_OWNERSHIP")
		}
	}
	pi := writePiFixture(t, t.TempDir())
	life, p := ledgerAll(t, execute, pi, "d1")
	if p.DetailStatus != "unavailable" || len(life) != 1 || life[0]["kind"] != "lifecycle" || life[0]["usage"] != nil {
		t.Fatal("ASSERT_LEDGER_LIFECYCLE")
	}
	sdk := writeSDKCLIFixture(t, t.TempDir())
	child, _ := ledgerAll(t, execute, sdk, "aaa")
	if len(child) != 2 || child[0]["usage"].(map[string]any)["known_total_tokens"] != float64(30) {
		t.Fatal("ASSERT_LEDGER_SDK")
	}
	cc := writeClaudeCodeFixture(t, t.TempDir())
	root, _ := ledgerAll(t, execute, cc, "ROOT")
	if len(root) != 5 || root[4]["tools"].(map[string]any)["match_status"] != "matched" {
		t.Fatal("ASSERT_LEDGER_CLAUDE")
	}
	missing, p := ledgerAll(t, execute, cc, "toolu_cc")
	if len(missing) != 0 || p.DetailStatus != "unavailable" {
		t.Fatal("ASSERT_LEDGER_NO_INFERRED_CHILD")
	}
	if _, err := execute("transcript", "events", sdk, "../../outside"); err == nil {
		t.Fatal("ASSERT_LEDGER_PATH")
	}
	file := filepath.Join(strings.TrimSuffix(sdk, ".jsonl"), "subagents", "agent-escape.jsonl")
	if err := os.Symlink(sdk, file); err != nil {
		t.Fatal(err)
	}
	if _, err := execute("transcript", "events", sdk, "escape"); err == nil {
		t.Fatal("ASSERT_LEDGER_PATH")
	}
}

func TestTranscriptEventsPagination(t *testing.T) {
	parent, _ := nativeToolResultFixture(t, true)
	_, execute := setupNotebook(t)
	events, p := ledgerAll(t, execute, parent, "AAA", "--payload")
	if p.Pages < 2 || len(events) != 3 {
		t.Fatal("ASSERT_LEDGER_LOSSLESS")
	}
	payload := events[1]["payload"].(map[string]any)
	blocks := payload["content"].([]any)
	want := "OWNED_NATIVE_RESULT " + strings.Repeat("α<&😀", 9000)
	if blocks[0].(map[string]any)["text"] != want {
		t.Fatal("ASSERT_LEDGER_LOSSLESS")
	}
	for _, args := range [][]string{
		{"--page", "2"}, {"--page", "0"}, {"--select", "bogus"}, {"--page", "2", "--snapshot", p.Snapshot},
		{"--payload", "--page", "99999", "--snapshot", p.Snapshot}, {"--payload", "--snapshot", "bad"},
	} {
		if _, err := execute(append([]string{"transcript", "events", parent, "AAA"}, args...)...); err == nil {
			t.Fatalf("ASSERT_LEDGER_REJECT: %v", args)
		}
	}
	_, side := nativeToolResultFixture(t, true)
	// Mutating selected payload must reject a pinned projection, even with unchanged IDs.
	source, err := os.ReadFile(side)
	if err != nil {
		t.Fatal(err)
	}
	_, q := ledgerAll(t, execute, side, "AAA", "--payload")
	if err := os.WriteFile(side, []byte(strings.Replace(string(source), "VISIBLE_ASSISTANT", "CHANGED_ASSISTANT", 1)), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := execute("transcript", "events", side, "AAA", "--payload", "--snapshot", q.Snapshot); err == nil {
		t.Fatal("ASSERT_LEDGER_STALE")
	}
}

func TestTranscriptEventsAmbiguousJoin(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pi.jsonl")
	writeTranscriptFile(t, path, `{"type":"session"}`+"\n"+`{"type":"message","message":{"role":"assistant","content":[{"type":"toolCall","id":"x","name":"bash"},{"type":"toolCall","id":"x","name":"bash"},{"type":"toolCall","name":"bash"}]}}`+"\n"+`{"type":"toolResult","message":{"role":"toolResult","toolCallId":"x","content":"ok"}}`+"\n")
	_, execute := setupNotebook(t)
	events, _ := ledgerAll(t, execute, path, "ROOT")
	for _, i := range []int{1, 2, 5} {
		if events[i]["tools"].(map[string]any)["match_status"] != "ambiguous" {
			t.Fatal("ASSERT_LEDGER_AMBIGUOUS")
		}
	}
	if events[3]["tools"].(map[string]any)["match_status"] != "unavailable" {
		t.Fatal("ASSERT_LEDGER_AMBIGUOUS")
	}
}

func TestTranscriptEventsUsageValidation(t *testing.T) {
	for _, raw := range []string{`{"input":-1}`, `{"input":1.2}`, `{"input":9223372036854775807,"output":1}`, `{"input":"3"}`, `[]`} {
		if _, err := ledgerUsage(json.RawMessage(raw)); err == nil {
			t.Fatalf("ASSERT_LEDGER_INVALID_USAGE: %s", raw)
		}
	}
	u, err := ledgerUsage(nil)
	if err != nil || u["status"] != "unavailable" {
		t.Fatal("ASSERT_LEDGER_UNKNOWN_ZERO")
	}
}
