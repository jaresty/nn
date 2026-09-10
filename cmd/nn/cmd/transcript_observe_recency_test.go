package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func recencyFixture(t *testing.T, n int, known bool) string {
	t.Helper()
	dir := t.TempDir()
	var lines []string
	emit := func(v map[string]any) {
		if known {
			v["timestamp"] = "2000-01-01T00:00:00Z"
		}
		b, e := json.Marshal(v)
		if e != nil {
			t.Fatal(e)
		}
		lines = append(lines, string(b))
	}
	emit(map[string]any{"type": "session", "version": 3, "id": "recency"})
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("worker-%03d", i)
		path := filepath.Join(dir, "pi-subagents-test", "session", "tasks", id+".output")
		writeTranscriptFile(t, path, fmt.Sprintf(`{"type":"assistant","agentId":%q,"isSidechain":true,"timestamp":"2000-01-01T00:00:00Z","message":{"role":"assistant","content":[{"type":"text","text":"OLD_PRIVATE_HISTORY"}]}}`+"\n", id))
		old := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
		if e := os.Chtimes(path, old, old); e != nil {
			t.Fatal(e)
		}
		emit(map[string]any{"type": "message", "id": "launch-" + id, "message": map[string]any{"role": "assistant", "content": []any{map[string]any{"type": "toolCall", "id": "call-" + id, "name": "Agent", "arguments": map[string]any{}}}}})
		emit(map[string]any{"type": "message", "id": "ack-" + id, "parentId": "launch-" + id, "message": map[string]any{"role": "toolResult", "toolName": "Agent", "toolCallId": "call-" + id, "details": map[string]any{"status": "background", "agentId": id}, "content": []any{map[string]any{"type": "text", "text": "Output file: " + path}}}})
	}
	path := filepath.Join(dir, "session.jsonl")
	writeTranscriptFile(t, path, strings.Join(lines, "\n")+"\n")
	return path
}
func TestObserveOldWorkerNotOpened(t *testing.T) {
	path := recencyFixture(t, 1, true)
	before := transcriptDecodeCount.Load()
	_, state, e := observeLiveSection(path, observeAttentionOptions{Limit: 1}, 5*time.Minute, "")
	if e != nil {
		t.Fatal(e)
	}
	if state.WorkerScans["metadata_old"] != 1 || state.WorkerScans["fresh_stream"] != 0 || transcriptDecodeCount.Load()-before != 3 {
		t.Fatalf("OLD_WORKER_READ FAIL: scans=%v decoded=%d", state.WorkerScans, transcriptDecodeCount.Load()-before)
	}
	t.Log("OLD_WORKER_READ PASS; PARENT_MTIME_INDEPENDENCE PASS")
	_, execute := setupNotebook(t)
	out, e := execute("transcript", "observe", path)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(out, "OLD_PRIVATE_HISTORY") || !strings.Contains(out, "History not inspected: outside_window") {
		t.Fatal("OLD_READABLE_HISTORY FAIL", out)
	}
}
func TestObserveUnknownRecencyBudget(t *testing.T) {
	path := recencyFixture(t, 26, false)
	text, state, e := observeLiveSection(path, observeAttentionOptions{Limit: 1}, 5*time.Minute, "")
	if e != nil {
		t.Fatal(e)
	}
	deferred := 0
	for _, a := range state.Agents {
		if a.State == "deferred" {
			deferred++
		}
	}
	if deferred != 6 || state.WorkerScans["fresh_stream"] != 20 || !strings.Contains(text, "deferred=6") {
		t.Fatalf("UNKNOWN_BUDGET FAIL: deferred=%d scans=%v", deferred, state.WorkerScans)
	}
	state.Text = "header\n## ROOT — retained body"
	state.ReadableOffset = len("header")
	id, e := saveObserveState(*state)
	if e != nil {
		t.Fatal(e)
	}
	before := transcriptDecodeCount.Load()
	retained, reused, e := tryObserveUnchanged(path, observeAttentionOptions{Limit: 1}, 5*time.Minute, id)
	if e != nil || !reused || transcriptDecodeCount.Load() != before || !strings.Contains(retained.Text, "deferred=6") {
		t.Fatalf("DEFERRED_REFRESH FAIL: reused=%v error=%v", reused, e)
	}
	t.Log("UNKNOWN_BUDGET PASS; DEFERRED_REFRESH PASS")
}
func TestObserveSourceChangeClock(t *testing.T) {
	cutoff := time.Now().Add(-5 * time.Minute)
	old := cutoff.Add(-time.Hour)
	stamp := observeSourceStamp{Stable: true, Modified: old.UnixNano()}
	if observeWorkerRecency(stamp, nil, time.Now(), true, cutoff) != "recent" {
		t.Fatal("RECENT_PARENT FAIL")
	}
	stamp.Modified = time.Now().UnixNano()
	if observeWorkerRecency(stamp, nil, old, false, cutoff) != "recent" {
		t.Fatal("RECENT_SOURCE FAIL")
	}
}
