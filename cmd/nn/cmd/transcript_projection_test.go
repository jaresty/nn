package cmd

import (
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

// Exercise the public command, checking the transport independently of its packer.
func reconstructTranscriptProjection(t *testing.T, execute func(...string) (string, error), session, agent string, raw bool) (string, string) {
	t.Helper()
	args := []string{"transcript", "show", session, agent}
	mode := "meaningful"
	if raw {
		args = append(args, "--raw")
		mode = "raw"
	}
	legacy, err := execute(args...)
	if err != nil {
		t.Fatal(err)
	}
	var rebuilt strings.Builder
	snapshot := ""
	pages, ordinal, total := 1, 1, 0
	for page := 1; page <= pages; page++ {
		paged := append(append([]string(nil), args...), "--json", "--page", strconv.Itoa(page))
		if snapshot != "" {
			paged = append(paged, "--snapshot", snapshot)
		}
		out, err := execute(paged...)
		if err != nil {
			t.Fatalf("projection page %d: %v", page, err)
		}
		if len(out) > 48000 || !utf8.ValidString(out) {
			t.Fatalf("invalid page transport: %d bytes", len(out))
		}
		var got transcriptShowTestPage
		if err := json.Unmarshal([]byte(out), &got); err != nil {
			t.Fatal(err)
		}
		if page == 1 {
			snapshot, pages = got.Snapshot, got.Pages
			if len(snapshot) != 64 || pages < 1 || len(got.Segments) == 0 {
				t.Fatal("invalid initial pagination envelope")
			}
			total = got.Segments[0].Segments
		}
		next := page + 1
		if page == pages {
			next = 0
		}
		if got.Page != page || got.Pages != pages || got.Snapshot != snapshot || got.Mode != mode || got.NextPage != next {
			t.Fatal("inconsistent pagination envelope")
		}
		for _, s := range got.Segments {
			if s.Segment != ordinal || s.Segments != total || !utf8.ValidString(s.Text) {
				t.Fatal("inconsistent segment stream")
			}
			ordinal++
			rebuilt.WriteString(s.Text)
		}
	}
	if ordinal-1 != total || rebuilt.String() != legacy {
		t.Fatal("projection reconstruction differs from text output")
	}
	return rebuilt.String(), snapshot
}

func TestTranscriptProjectionSidechainOwnership(t *testing.T) {
	const assertion = "ASSERT_PI_SHOW_FILTERS_SIDECAR_EVENT_OWNERSHIP"
	for _, terminal := range []bool{false, true} {
		for _, matching := range []bool{false, true} {
			for _, raw := range []bool{false, true} {
				name := "terminal=" + strconv.FormatBool(terminal) + "/matching=" + strconv.FormatBool(matching) + "/raw=" + strconv.FormatBool(raw)
				t.Run(name, func(t *testing.T) {
					dir := t.TempDir()
					side := filepath.Join(dir, "pi-subagents-test", "session", "tasks", "AAA.output")
					bad := `{"type":"assistant","agentId":"BBB","message":{"role":"assistant","content":"FOREIGN_EVENT"}}` + "\n" +
						`{"type":"assistant","message":{"role":"assistant","content":"UNOWNED_EVENT"}}` + "\n" +
						`{"type":"custom","agentId":"AAA","message":{"role":"assistant","content":"NON_EVENT"}}` + "\n"
					content := bad
					if matching {
						for _, kind := range []string{"user", "assistant", "message"} {
							content += `{"type":"` + kind + `","agentId":"AAA","message":{"role":"assistant","content":"OWNED_` + kind + `"}}` + "\n"
						}
					}
					writeTranscriptFile(t, side, content)
					session := filepath.Join(dir, "parent.jsonl")
					parent := `{"type":"session"}` + "\n" +
						`{"type":"message","message":{"role":"assistant","content":"ROOT_MESSAGE"}}` + "\n" +
						`{"type":"message","message":{"role":"toolResult","details":{"status":"background","agentId":"AAA","fullOutputPath":` + mustJSONString(t, side) + `}}}` + "\n"
					if terminal {
						parent += `{"type":"custom","customType":"subagents:record","data":{"id":"AAA","type":"general-purpose","status":"completed","result":"TERMINAL_FALLBACK"}}` + "\n"
					}
					writeTranscriptFile(t, session, parent)
					_, execute := setupNotebook(t)
					out, snapshot := reconstructTranscriptProjection(t, execute, session, "AAA", raw)
					for _, forbidden := range []string{"FOREIGN_EVENT", "UNOWNED_EVENT", "NON_EVENT", "ROOT_MESSAGE"} {
						if strings.Contains(out, forbidden) {
							t.Fatalf("%s: rendered %s", assertion, forbidden)
						}
					}
					if matching {
						for _, kind := range []string{"user", "assistant", "message"} {
							if strings.Count(out, "OWNED_"+kind) != 1 {
								t.Fatalf("%s: missing or duplicated owned %s", assertion, kind)
							}
						}
						if strings.Contains(out, "TERMINAL_FALLBACK") {
							t.Fatalf("%s: duplicate terminal detail", assertion)
						}
						// Removing the owned evidence must invalidate the projection snapshot.
						writeTranscriptFile(t, side, bad)
						args := []string{"transcript", "show", session, "AAA", "--json", "--snapshot", snapshot}
						if raw {
							args = append(args, "--raw")
						}
						if _, err := execute(args...); err == nil || !strings.Contains(err.Error(), "stale or mismatched") {
							t.Fatalf("%s: evidence removal did not invalidate snapshot: %v", assertion, err)
						}
					} else if terminal {
						if !strings.Contains(out, "TERMINAL_FALLBACK") || !strings.Contains(out, "status: completed") {
							t.Fatalf("%s: missing terminal fallback", assertion)
						}
					} else if !strings.Contains(out, "sidechain output unavailable for AAA") {
						t.Fatalf("%s: missing provisional fallback", assertion)
					}
					root, _ := reconstructTranscriptProjection(t, execute, session, "ROOT", raw)
					if !strings.Contains(root, "ROOT_MESSAGE") {
						t.Fatalf("%s: main-stream ROOT convention changed", assertion)
					}
				})
			}
		}
	}
}

func TestTranscriptProjectionRuneSafePreviews(t *testing.T) {
	const assertion = "ASSERT_TOOL_PREVIEW_PRESERVES_UTF8_BOUNDARIES"
	for _, runeText := range []string{"a", "α", "界", "😀"} {
		for _, offset := range []int{0, 1, 2, 3} {
			t.Run(runeText+strconv.Itoa(offset), func(t *testing.T) {
				// Seven bytes of JSON prefix place the first rune around byte 120.
				input := `{"x": "` + strings.Repeat("a", 120-7-offset) + strings.Repeat(runeText, 10) + `"}`
				content := `[{"type":"toolCall","name":"probe","input":` + input + `}]`
				got := textContent(json.RawMessage(content))
				end := 120
				for end > 0 && !utf8.ValidString(input[:end]) {
					end--
				}
				want := "→ probe(" + input[:end] + "…)"
				if got != want || !utf8.ValidString(got) {
					t.Fatalf("%s: got %q, want %q", assertion, got, want)
				}
				session := filepath.Join(t.TempDir(), "direct.output")
				writeTranscriptFile(t, session, `{"isSidechain":true,"agentId":"AAA","type":"assistant","message":{"role":"assistant","content":`+content+`}}`+"\n")
				_, execute := setupNotebook(t)
				for _, raw := range []bool{false, true} {
					reconstructTranscriptProjection(t, execute, session, "AAA", raw)
				}
			})
		}
	}
}

func TestTranscriptProjectionEncodingBoundary(t *testing.T) {
	_, execute := setupNotebook(t)
	session := filepath.Join(t.TempDir(), "invalid.output")
	prefix := `{"isSidechain":true,"agentId":"AAA","type":"assistant","message":{"role":"assistant","content":"`
	suffix := `"}}` + "\n"
	writeTranscriptFile(t, session, prefix+string([]byte{0xff})+suffix)
	out, snapshot := reconstructTranscriptProjection(t, execute, session, "AAA", false)
	if !strings.Contains(out, "�") {
		t.Fatal("decoded projection must retain the JSON decoder's replacement character")
	}
	writeTranscriptFile(t, session, prefix+string([]byte{0xfe})+suffix)
	_, next := reconstructTranscriptProjection(t, execute, session, "AAA", false)
	if next != snapshot {
		t.Fatal("same request and projected output must retain snapshot despite source-byte differences")
	}
	if _, err := execute("transcript", "show", session, "AAA", "--json", "--snapshot", snapshot); err != nil {
		t.Fatal(err)
	}

	// SDK raw mode really is a byte-preserving projection, unlike decoded text.
	sdk := writeSDKCLIFixture(t, t.TempDir())
	detail := filepath.Join(strings.TrimSuffix(sdk, ".jsonl"), "subagents", "agent-aaa.jsonl")
	valid := `{"type":"user","message":{"role":"user","content":"` + strings.Repeat("α<&😀", 9000) + `"}}` + "\n" +
		`{"type":"attachment","attachment":{"text":"RAW_ONLY"}}` + "\n"
	writeTranscriptFile(t, detail, valid)
	meaningful, _ := reconstructTranscriptProjection(t, execute, sdk, "aaa", false)
	raw, _ := reconstructTranscriptProjection(t, execute, sdk, "aaa", true)
	if strings.Contains(meaningful, "RAW_ONLY") || !strings.Contains(raw, "RAW_ONLY") {
		t.Fatal("SDK fixture must exercise genuinely distinct raw and meaningful payloads")
	}
	writeTranscriptFile(t, detail, valid+string([]byte{0xff}))
	legacy, err := execute("transcript", "show", sdk, "aaa", "--raw")
	if err != nil || utf8.ValidString(legacy) {
		t.Fatalf("SDK raw text must retain invalid bytes: %v", err)
	}
	if _, err := execute("transcript", "show", sdk, "aaa", "--raw", "--json"); err == nil || !strings.Contains(err.Error(), "projected output is not valid UTF-8") {
		t.Fatalf("JSON must reject invalid projected UTF-8: %v", err)
	}
}
