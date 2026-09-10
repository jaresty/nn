package cmd

import (
	"reflect"
	"testing"
	"time"
)

func TestObserveParentRoster(t *testing.T) {
	path := writePiBackgroundSidechainFixture(t, t.TempDir())
	agents, err := buildTree(path)
	if err != nil {
		t.Fatal(err)
	}
	native, err := buildTreeChildPage(agents, "ROOT", 2, "")
	if err != nil {
		t.Fatal(err)
	}
	before := transcriptDecodeCount.Load()
	roster, parent, err := observeRoster(path)
	if err != nil {
		t.Fatal(err)
	}
	if parent == nil {
		t.Fatal("ROSTER_PARENT FAIL")
	}
	if got, want := transcriptDecodeCount.Load()-before, uint64(len(parent.Sources[parent.Path].Records)); got != want {
		t.Fatalf("ROSTER_NO_WORKER_READ FAIL: %d != %d", got, want)
	}
	if roster.TotalChildren != native.TotalChildren || roster.Omitted != native.Omitted || len(roster.Children) != len(native.Children) {
		t.Fatal("ROSTER_PARITY FAIL")
	}
	for i, a := range roster.Children {
		b := native.Children[i]
		if a.ID != b.ID || a.ParentID != b.ParentID || a.Description != b.Description || a.ParentageStatus != b.ParentageStatus {
			t.Fatal("ROSTER_PARITY FAIL", a, b)
		}
	}
	t.Log("ROSTER_PARITY PASS; ROSTER_NO_WORKER_READ PASS")
}
func TestObserveSharedParent(t *testing.T) {
	path := observeAttentionFixture(t)
	_, parent, err := observeRoster(path)
	if err != nil {
		t.Fatal(err)
	}
	before := transcriptDecodeCount.Load()
	_, shared, err := observeLiveSection(path, observeAttentionOptions{Limit: 20}, 5*time.Minute, "", parent)
	if err != nil {
		t.Fatal(err)
	}
	if transcriptDecodeCount.Load() != before {
		t.Fatal("ROSTER_SHARED_PARENT FAIL: parent decoded again")
	}
	_, fresh, err := observeLiveSection(path, observeAttentionOptions{Limit: 20}, 5*time.Minute, "")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(shared.Agents, fresh.Agents) {
		t.Fatal("ROSTER_SHARED_PARENT FAIL: different results")
	}
	t.Log("ROSTER_SHARED_PARENT PASS")
}
