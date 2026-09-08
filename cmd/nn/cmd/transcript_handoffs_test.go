package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func handoffFixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "session.jsonl")
	writeTranscriptFile(t, path, strings.Join([]string{
		`{"type":"session"}`,
		`{"type":"message","id":"invoke","message":{"role":"assistant","content":[{"type":"toolCall","id":"c1","name":"Agent","arguments":{"description":"Run twelve seeds","prompt":"EXACT ASSIGNMENT"}}]}}`,
		`{"type":"message","id":"ack","parentId":"invoke","message":{"role":"toolResult","toolName":"Agent","toolCallId":"c1","details":{"status":"background","agentId":"AAA","description":"Run twelve seeds","subagentType":"general-purpose"}}}`,
		`{"type":"custom","id":"done1","customType":"subagents:record","data":{"id":"AAA","type":"general-purpose","status":"completed","result":"first return"}}`,
		`{"type":"message","id":"invoke2","message":{"role":"assistant","content":[{"type":"toolCall","id":"c2","name":"Agent","arguments":{"resume":"AAA","description":"Resume trial","prompt":"SECOND ASSIGNMENT"}}]}}`,
		`{"type":"message","id":"ack2","parentId":"invoke2","message":{"role":"toolResult","toolName":"Agent","toolCallId":"c2","details":{"status":"background","agentId":"AAA","subagentType":"general-purpose"}}}`,
		`{"type":"custom","id":"done2","customType":"subagents:record","data":{"id":"AAA","type":"general-purpose","status":"error","result":"second return"}}`,
		`{"type":"message","id":"other","message":{"role":"toolResult","toolName":"Agent","toolCallId":"missing","details":{"status":"background","agentId":"BBB","description":"Other child"}}}`,
	}, "\n")+"\n")
	return path
}

func TestTranscriptHandoffDescription(t *testing.T) {
	const a = "ASSERT_HANDOFF_DESCRIPTION"
	path := handoffFixture(t)
	rows, err := buildTree(path)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(rows)
	var decoded []map[string]any
	_ = json.Unmarshal(b, &decoded)
	found := false
	for _, row := range decoded {
		if row["id"] == "AAA" {
			found = row["description"] == "Resume trial"
		}
	}
	if !found || !strings.Contains(renderOverview(rows), `"Resume trial"`) {
		t.Fatalf("%s: %s", a, b)
	}
	projection, err := projectTranscriptTree(rows, "AAA", "id,description")
	if err != nil {
		t.Fatalf("%s: %v", a, err)
	}
	b, _ = json.Marshal(projection)
	if !strings.Contains(string(b), "Resume trial") {
		t.Fatalf("%s: selected field absent", a)
	}
	t.Log(a + ": PASS")
}

func TestTranscriptHandoffSelection(t *testing.T) {
	const a = "ASSERT_HANDOFF_SELECTION"
	path := handoffFixture(t)
	_, execute := setupNotebook(t)
	for _, kind := range []string{"launch", "return"} {
		out, err := execute("transcript", "events", path, "AAA", "--at", kind, "--payload")
		if err != nil {
			t.Fatalf("%s: %v", a, err)
		}
		var p struct {
			Events  []map[string]any `json:"events"`
			Handoff map[string]any   `json:"handoff"`
		}
		_ = json.Unmarshal([]byte(out), &p)
		if len(p.Events) != 2 || p.Handoff["status"] != "observed" || p.Handoff["occurrences"] != float64(2) {
			t.Fatalf("%s: %s", a, out)
		}
		for i, e := range p.Events {
			if e["kind"] != kind || e["occurrence"] != float64(i+1) || e["agent_id"] != "AAA" || e["source"].(map[string]any)["path"] == "" {
				t.Fatalf("%s: invalid identity: %s", a, out)
			}
		}
		if kind == "launch" {
			m := p.Events[0]["lifecycle"].(map[string]any)
			if m["match_status"] != "matched" || !strings.Contains(out, "EXACT ASSIGNMENT") || strings.Contains(out, "Other child") {
				t.Fatalf("%s: invocation join: %s", a, out)
			}
		}
	}
	out, err := execute("transcript", "events", path, "BBB", "--at", "return")
	if err != nil || !strings.Contains(out, `"status":"not_observed"`) {
		t.Fatalf("%s: no return: %v %s", a, err, out)
	}
	out, err = execute("transcript", "events", path, "BBB", "--at", "launch", "--payload")
	if err != nil || !strings.Contains(out, `"match_status":"missing"`) {
		t.Fatalf("%s: missing invocation: %v %s", a, err, out)
	}
	// Duplicate invocation IDs must not select an arbitrary prompt.
	b, _ := os.ReadFile(path)
	writeTranscriptFile(t, path, string(b)+`{"type":"message","message":{"role":"assistant","content":[{"type":"toolCall","id":"c1","name":"Agent","arguments":{"prompt":"AMBIGUOUS PROMPT"}}]}}`+"\n")
	out, err = execute("transcript", "events", path, "AAA", "--at", "launch", "--payload")
	if err != nil || !strings.Contains(out, `"match_status":"ambiguous"`) || strings.Contains(out, "EXACT ASSIGNMENT") || strings.Contains(out, "AMBIGUOUS PROMPT") {
		t.Fatalf("%s: ambiguous invocation: %v %s", a, err, out)
	}
	for _, flags := range [][]string{{"--at", "bad"}, {"--at", "launch", "--summary", "tools"}, {"--at", "return", "--errors-only"}, {"--at", "launch", "--since", "2026-09-08T00:00:00Z"}} {
		if _, err := execute(append([]string{"transcript", "events", path, "AAA"}, flags...)...); err == nil {
			t.Fatalf("%s: accepted %v", a, flags)
		}
	}
	if _, err := execute("transcript", "events", path, "UNKNOWN", "--at", "return"); err == nil {
		t.Fatalf("%s: unknown child accepted", a)
	}
	// Same call ID in another owner's stream is not a candidate invocation.
	fresh := handoffFixture(t)
	bytes, _ := os.ReadFile(fresh)
	writeTranscriptFile(t, fresh, string(bytes)+`{"type":"message","agentId":"OTHER","message":{"role":"assistant","content":[{"type":"toolCall","id":"c1","name":"Agent","arguments":{"prompt":"FOREIGN PROMPT"}}]}}`+"\n"+`{"type":"message","message":{"role":"user","toolName":"Agent","details":{"status":"background","agentId":"AAA","description":"SPOOF"}}}`+"\n")
	out, err = execute("transcript", "events", fresh, "AAA", "--at", "launch", "--payload")
	if err != nil || !strings.Contains(out, "EXACT ASSIGNMENT") || strings.Contains(out, "FOREIGN PROMPT") || strings.Contains(out, "SPOOF") {
		t.Fatalf("%s: owner/role scope: %v %s", a, err, out)
	}
	var launch ledgerPage
	_ = json.Unmarshal([]byte(out), &launch)
	var first map[string]any
	_ = json.Unmarshal(launch.Events[0], &first)
	out, err = execute("transcript", "events", fresh, "AAA", "--at", "launch", "--event", first["event_id"].(string), "--payload")
	var exact ledgerPage
	_ = json.Unmarshal([]byte(out), &exact)
	if err != nil || len(exact.Events) != 1 || string(exact.Events[0]) != string(launch.Events[0]) {
		t.Fatalf("%s: exact handoff retrieval", a)
	}
	unsupported := filepath.Join(t.TempDir(), "unknown.jsonl")
	writeTranscriptFile(t, unsupported, "{}\n")
	out, err = execute("transcript", "events", unsupported, "AAA", "--at", "launch")
	if err != nil || !strings.Contains(out, `"status":"unsupported_schema"`) {
		t.Fatalf("%s: unsupported schema: %v %s", a, err, out)
	}
	t.Log(a + ": PASS")
}

func TestTranscriptHandoffTransport(t *testing.T) {
	const a = "ASSERT_HANDOFF_TRANSPORT"
	path := handoffFixture(t)
	b, _ := os.ReadFile(path)
	writeTranscriptFile(t, path, strings.Replace(string(b), "EXACT ASSIGNMENT", strings.Repeat("α😀", 16000), 1))
	_, execute := setupNotebook(t)
	flags := []string{"transcript", "events", path, "AAA", "--at", "launch", "--payload"}
	out, err := execute(flags...)
	if err != nil {
		t.Fatalf("%s: %v", a, err)
	}
	var page ledgerPage
	_ = json.Unmarshal([]byte(out), &page)
	snapshot := page.Snapshot
	if page.Pages < 2 {
		t.Fatalf("%s: missing fragmentation", a)
	}
	var partial strings.Builder
	var entries []string
	for {
		if len(out) > 48000 {
			t.Fatalf("%s: oversized page", a)
		}
		for _, raw := range page.Events {
			var f struct {
				Text     string `json:"text"`
				Segment  int    `json:"segment"`
				Segments int    `json:"segments"`
			}
			_ = json.Unmarshal(raw, &f)
			if f.Segment > 0 {
				partial.WriteString(f.Text)
				if f.Segment == f.Segments {
					entries = append(entries, partial.String())
					partial.Reset()
				}
			} else {
				entries = append(entries, string(raw))
			}
		}
		if page.NextPage == 0 {
			break
		}
		out, err = execute(append(flags, "--page", fmt.Sprint(page.NextPage), "--snapshot", snapshot)...)
		if err != nil {
			t.Fatalf("%s: %v", a, err)
		}
		_ = json.Unmarshal([]byte(out), &page)
	}
	out, err = execute(append(flags, "--all")...)
	if err != nil {
		t.Fatalf("%s: %v", a, err)
	}
	_ = json.Unmarshal([]byte(out), &page)
	if len(entries) != 2 || page.Snapshot != snapshot || len(page.Events) != 2 || string(page.Events[0]) != entries[0] || string(page.Events[1]) != entries[1] {
		t.Fatalf("%s: lossy reassembly", a)
	}
	if _, err := execute("transcript", "events", path, "AAA", "--at", "return", "--payload", "--snapshot", snapshot); err == nil {
		t.Fatalf("%s: cross-direction snapshot accepted", a)
	}
	changed, _ := os.ReadFile(path)
	writeTranscriptFile(t, path, strings.Replace(string(changed), "SECOND ASSIGNMENT", "CHANGED ASSIGNMENT", 1))
	if _, err := execute(append(flags, "--snapshot", snapshot)...); err == nil {
		t.Fatalf("%s: stale invocation accepted", a)
	}
	t.Log(a + ": PASS")
}

func TestTranscriptHandoffSkill(t *testing.T) {
	const a = "ASSERT_HANDOFF_SKILL"
	for _, path := range []string{"references/handoffs.md"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "..", "skills", "nn-transcript", path))
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range []string{"--at launch", "--at return", "description", "occurrence"} {
			if !strings.Contains(string(b), s) {
				t.Fatalf("%s: %s lacks %s", a, path, s)
			}
		}
	}
	t.Log(a + ": PASS")
}
