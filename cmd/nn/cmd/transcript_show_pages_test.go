package cmd

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

type transcriptShowTestSegment struct {
	Segment  int    `json:"segment"`
	Segments int    `json:"segments"`
	Text     string `json:"text"`
}

type transcriptShowTestPage struct {
	Snapshot string                      `json:"snapshot"`
	Page     int                         `json:"page"`
	Pages    int                         `json:"pages"`
	NextPage int                         `json:"next_page"`
	Mode     string                      `json:"mode"`
	Segments []transcriptShowTestSegment `json:"segments"`
}

func TestTranscriptShowJSONPagesAreBoundedLosslessAndSnapshotBound(t *testing.T) {
	const assertion = "ASSERT_TRANSCRIPT_SHOW_PAGES_ARE_BOUNDED_LOSSLESS_AND_SNAPSHOT_BOUND"
	dir := t.TempDir()
	session := filepath.Join(dir, "large.output")
	large := strings.Repeat("α<&\n", 12000)
	writeTranscriptFile(t, session,
		`{"isSidechain":true,"agentId":"AAA","type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":`+mustJSONString(t, large)+`}]}}`+"\n")
	_, execute := setupNotebook(t)
	legacy, err := execute("transcript", "show", session, "AAA")
	if err != nil {
		t.Fatal(err)
	}

	firstOut, err := execute("transcript", "show", session, "AAA", "--json")
	if err != nil {
		t.Fatalf("%s: page 1: %v", assertion, err)
	}
	var first transcriptShowTestPage
	if err := json.Unmarshal([]byte(firstOut), &first); err != nil {
		t.Fatalf("%s: page 1 JSON: %v", assertion, err)
	}
	if len(firstOut) > 48000 || first.Pages < 2 || len(first.Snapshot) != 64 || first.Mode != "meaningful" {
		t.Fatalf("%s: invalid first page bytes/pages/snapshot/mode = %d/%d/%q/%q", assertion, len(firstOut), first.Pages, first.Snapshot, first.Mode)
	}

	if _, err := execute("transcript", "show", session, "AAA", "--json", "--page", "2"); err == nil || !strings.Contains(err.Error(), "--snapshot is required") {
		t.Fatalf("%s: missing later-page snapshot error = %v", assertion, err)
	}

	var rebuilt strings.Builder
	expectedSegment, totalSegments := 1, 0
	for page := 1; page <= first.Pages; page++ {
		args := []string{"transcript", "show", session, "AAA", "--json", "--page", itoa(page)}
		if page > 1 {
			args = append(args, "--snapshot", first.Snapshot)
		}
		out, err := execute(args...)
		if err != nil {
			t.Fatalf("%s: page %d: %v", assertion, page, err)
		}
		if len(out) > 48000 {
			t.Fatalf("%s: page %d is %d bytes", assertion, page, len(out))
		}
		var got transcriptShowTestPage
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatal(err)
		}
		wantNext := 0
		if page < first.Pages {
			wantNext = page + 1
		}
		if got.Snapshot != first.Snapshot || got.Page != page || got.Pages != first.Pages || got.NextPage != wantNext {
			t.Fatalf("%s: inconsistent page %d envelope", assertion, page)
		}
		for _, segment := range got.Segments {
			if segment.Segment != expectedSegment {
				t.Fatalf("%s: segment ordinal = %d, want %d", assertion, segment.Segment, expectedSegment)
			}
			totalSegments = segment.Segments
			expectedSegment++
			rebuilt.WriteString(segment.Text)
		}
	}
	if rebuilt.String() != legacy || expectedSegment-1 != totalSegments {
		t.Fatalf("%s: reconstruction differs or is incomplete: got %d bytes/%d segments want %d bytes/%d segments", assertion, rebuilt.Len(), expectedSegment-1, len(legacy), totalSegments)
	}

	rawLegacy, err := execute("transcript", "show", session, "AAA", "--raw")
	if err != nil {
		t.Fatal(err)
	}
	rawFirstOut, err := execute("transcript", "show", session, "AAA", "--json", "--raw")
	if err != nil {
		t.Fatal(err)
	}
	var rawFirst transcriptShowTestPage
	if err := json.Unmarshal([]byte(rawFirstOut), &rawFirst); err != nil {
		t.Fatal(err)
	}
	if rawFirst.Mode != "raw" || rawFirst.Snapshot == first.Snapshot {
		t.Fatalf("%s: raw mode/snapshot = %q/%q", assertion, rawFirst.Mode, rawFirst.Snapshot)
	}
	var rawRebuilt strings.Builder
	for page := 1; page <= rawFirst.Pages; page++ {
		args := []string{"transcript", "show", session, "AAA", "--json", "--raw", "--page", itoa(page)}
		if page > 1 {
			args = append(args, "--snapshot", rawFirst.Snapshot)
		}
		out, err := execute(args...)
		if err != nil {
			t.Fatal(err)
		}
		var got transcriptShowTestPage
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatal(err)
		}
		for _, segment := range got.Segments {
			rawRebuilt.WriteString(segment.Text)
		}
	}
	if rawRebuilt.String() != rawLegacy {
		t.Fatalf("%s: raw reconstruction differs: got %d bytes want %d", assertion, rawRebuilt.Len(), len(rawLegacy))
	}

	if first.Pages > 1 {
		writeTranscriptFile(t, session,
			`{"isSidechain":true,"agentId":"AAA","type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"changed"}]}}`+"\n")
		_, err := execute("transcript", "show", session, "AAA", "--json", "--page", "2", "--snapshot", first.Snapshot)
		if err == nil || !strings.Contains(err.Error(), "stale or mismatched") {
			t.Fatalf("%s: stale snapshot error = %v", assertion, err)
		}
	}
}

func mustJSONString(t *testing.T, value string) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func itoa(value int) string {
	data, _ := json.Marshal(value)
	return string(data)
}
