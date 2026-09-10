package cmd

import (
	"os"
	"testing"
	"time"
)

func TestObserveSkippedWorkerChangedRefresh(t *testing.T) {
	path := recencyFixture(t, 1, true)
	_, old, e := observeLiveSection(path, observeAttentionOptions{Limit: 1}, 5*time.Minute, "")
	if e != nil {
		t.Fatal(e)
	}
	snapshot, e := saveObserveState(*old)
	if e != nil {
		t.Fatal(e)
	}
	rows, e := readRecords(path)
	if e != nil {
		t.Fatal(e)
	}
	loc := piBackgroundLocators(rows)[0]
	f, e := os.OpenFile(loc.Path, os.O_APPEND|os.O_WRONLY, 0600)
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.WriteString(`{"type":"assistant","agentId":"worker-000","isSidechain":true,"timestamp":"2000-01-01T00:00:00Z","message":{"role":"assistant","content":[]}}` + "\n")
	closeErr := f.Close()
	if e != nil || closeErr != nil {
		t.Fatal(e, closeErr)
	}
	_, fresh, e := observeLiveSection(path, observeAttentionOptions{Limit: 1}, 5*time.Minute, snapshot)
	if e != nil {
		t.Fatal(e)
	}
	if fresh.WorkerScans["fresh_stream"] != 1 {
		t.Fatal("SKIPPED_CHANGED_REFRESH FAIL", fresh.WorkerScans)
	}
	for _, a := range fresh.Agents {
		if a.ID == "worker-000" {
			if a.Room == nil || a.Room.Window.Selected != 2 {
				t.Fatal("SKIPPED_CHANGED_REFRESH FAIL", a)
			}
			return
		}
	}
	t.Fatal("changed worker missing")
}
