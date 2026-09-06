package cmd

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

type cursorLSRow struct{ Session, Path, Modified, Cursor string }

func cursorLSFixture(t *testing.T) (string, []time.Time) {
	t.Helper()
	dir := t.TempDir()
	base := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	times := []time.Time{base.Add(900 * time.Millisecond), base.Add(500 * time.Millisecond), base.Add(123456789), base.Add(123456789)}
	for i, name := range []string{"new", "middle", "a", "b"} {
		path := filepath.Join(dir, name+".jsonl")
		writeTranscriptFile(t, path, "{\"type\":\"assistant\",\"message\":{\"role\":\"assistant\",\"content\":[]}}\n")
		if err := os.Chtimes(path, times[i], times[i]); err != nil {
			t.Fatal(err)
		}
	}
	return dir, times
}

func cursorLSPage(t *testing.T, dir string, args ...string) []cursorLSRow {
	t.Helper()
	_, execute := setupNotebook(t)
	out, err := execute(append([]string{"transcript", "ls", dir, "--json"}, args...)...)
	if err != nil {
		t.Fatal(err)
	}
	var rows []cursorLSRow
	if err := json.Unmarshal([]byte(out), &rows); err != nil {
		t.Fatal(err)
	}
	if rows == nil {
		t.Fatalf("expected JSON array: %s", out)
	}
	return rows
}

func TestTranscriptLsCursorTraversal(t *testing.T) {
	dir, times := cursorLSFixture(t)
	var got []string
	cursor := ""
	for i := 0; i < 5; i++ {
		args := []string{"--limit", "1"}
		if cursor != "" {
			args = append(args, "--cursor", cursor)
		}
		rows := cursorLSPage(t, dir, args...)
		if i == 4 {
			if len(rows) != 0 {
				t.Fatal("final page must be []")
			}
			break
		}
		if len(rows) != 1 {
			t.Fatalf("page %d: %+v", i, rows)
		}
		if rows[0].Cursor == "" {
			t.Fatal("ASSERT_TRANSCRIPT_LS_CURSOR_PRESENT: missing per-row cursor")
		}
		stamp, err := time.Parse(time.RFC3339Nano, rows[0].Modified)
		if err != nil || !stamp.Equal(times[i]) {
			t.Fatalf("nanoseconds lost: %s want %s", rows[0].Modified, times[i])
		}
		got = append(got, rows[0].Session)
		cursor = rows[0].Cursor
	}
	if !reflect.DeepEqual(got, []string{"new", "middle", "a", "b"}) {
		t.Fatalf("incomplete/duplicate/order: %v", got)
	}
}

func TestTranscriptLsCursorGuards(t *testing.T) {
	dir, times := cursorLSFixture(t)
	first := cursorLSPage(t, dir, "--limit", "1")[0]
	rows := cursorLSPage(t, dir+"/./", "--limit", "2", "--cursor", first.Cursor)
	if len(rows) != 2 || rows[0].Session != "middle" || rows[1].Session != "a" {
		t.Fatalf("changed limit/equivalent dir: %+v", rows)
	}
	before := times[1].Format(time.RFC3339Nano)
	filtered := cursorLSPage(t, dir, "--before", before, "--limit", "1")
	if len(filtered) != 1 || filtered[0].Session != "a" {
		t.Fatalf("strict before: %+v", filtered)
	}
	remaining := cursorLSPage(t, dir, "--before", times[1].In(time.FixedZone("offset", 3600)).Format(time.RFC3339Nano), "--cursor", filtered[0].Cursor)
	if len(remaining) != 1 || remaining[0].Session != "b" {
		t.Fatalf("filtered continuation: %+v", remaining)
	}
	fail := func(dir string, args ...string) {
		t.Helper()
		_, execute := setupNotebook(t)
		_, err := execute(append([]string{"transcript", "ls", dir, "--json"}, args...)...)
		if err == nil || !strings.Contains(err.Error(), "cursor") {
			t.Fatalf("expected explicit cursor error: %v", err)
		}
	}
	fail(t.TempDir(), "--cursor", first.Cursor)
	fail(dir, "--cursor", first.Cursor, "--before", before)
	fail(dir, "--cursor", filtered[0].Cursor)
	for _, raw := range []string{"", "%", base64.RawURLEncoding.EncodeToString([]byte("{")), base64.RawURLEncoding.EncodeToString([]byte("null"))} {
		fail(dir, "--cursor", raw)
	}
	payload, err := base64.RawURLEncoding.DecodeString(first.Cursor)
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []map[string]any{{"version": 99}, {"after": -1}, {"after": 4}, {"after": nil}, {"snapshot": "wrong"}} {
		var obj map[string]any
		if err := json.Unmarshal(payload, &obj); err != nil {
			t.Fatal(err)
		}
		for k, v := range change {
			obj[k] = v
		}
		b, _ := json.Marshal(obj)
		fail(dir, "--cursor", base64.RawURLEncoding.EncodeToString(b))
	}
	for _, kind := range []string{"addition", "removal", "mtime", "size"} {
		t.Run(kind, func(t *testing.T) {
			d, ts := cursorLSFixture(t)
			token := cursorLSPage(t, d, "--limit", "1")[0].Cursor
			p := filepath.Join(d, "b.jsonl")
			var err error
			switch kind {
			case "addition":
				err = os.WriteFile(filepath.Join(d, "extra.jsonl"), []byte("{}\n"), 0600)
			case "removal":
				err = os.Remove(p)
			case "mtime":
				err = os.Chtimes(p, ts[3].Add(time.Nanosecond), ts[3].Add(time.Nanosecond))
			case "size":
				err = os.WriteFile(p, []byte("{}\n"), 0600)
				if err == nil {
					err = os.Chtimes(p, ts[3], ts[3])
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			fail(d, "--cursor", token)
		})
	}
}
