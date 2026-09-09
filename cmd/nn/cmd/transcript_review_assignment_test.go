package cmd

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestReviewIncludesAssignments(t *testing.T) {
	_, execute := setupNotebook(t)
	path := handoffFixture(t)
	out, e := execute("transcript", "review", path, "--queue", "archive", "--order", "canonical", "--limit", "1", "--last", "2", "--include-assignment", "--format", "text")
	if e != nil {
		t.Fatalf("ASSERT_REVIEW_ASSIGNMENTS: native assignment bundle unavailable: %v", e)
	}
	for _, s := range []string{"EXACT ASSIGNMENT", "SECOND ASSIGNMENT", "LAUNCH", "steering: unavailable", "governing attempt: not_inferred", "assignment records: 2"} {
		if !strings.Contains(out, s) {
			t.Fatalf("ASSERT_REVIEW_ASSIGNMENTS: missing %q: %s", s, out)
		}
	}
	if strings.Contains(out, "Other child") {
		t.Fatal("ASSERT_REVIEW_ASSIGNMENTS: unselected room leaked")
	}
}

func TestReviewAssignmentValidation(t *testing.T) {
	_, execute := setupNotebook(t)
	path := handoffFixture(t)
	for _, flags := range [][]string{{"--include-assignment", "--json"}, {"--last", "2", "--include-assignment", "--payload=false", "--json"}} {
		if _, e := execute(append([]string{"transcript", "review", path}, flags...)...); e == nil {
			t.Fatalf("assignment flags accepted invalid request %v", flags)
		}
	}
}

func TestReviewAssignmentRetainedPaging(t *testing.T) {
	_, execute := setupNotebook(t)
	path := handoffFixture(t)
	b, e := os.ReadFile(path)
	if e != nil {
		t.Fatal(e)
	}
	b = []byte(strings.ReplaceAll(string(b), "EXACT ASSIGNMENT", strings.Repeat("BIG ASSIGNMENT ", 6000)))
	if e = os.WriteFile(path, b, 0600); e != nil {
		t.Fatal(e)
	}
	base := []string{"transcript", "review", path, "--queue", "archive", "--order", "canonical", "--limit", "1", "--last", "2"}
	out, e := execute(append(base, "--include-assignment", "--json")...)
	if e != nil {
		t.Fatal(e)
	}
	var first ledgerPage
	if e = json.Unmarshal([]byte(out), &first); e != nil {
		t.Fatal(e)
	}
	if first.Pages < 2 || !first.Payload {
		t.Fatal("ASSERT_ASSIGNMENT_PAGES: payload/fragmented fixture missing")
	}
	if e = os.Remove(path); e != nil {
		t.Fatal(e)
	}
	rendered, e := execute(append(base, "--include-assignment", "--format", "text", "--snapshot", first.Snapshot)...)
	if e != nil {
		t.Fatalf("ASSERT_ASSIGNMENT_REPLAY: %v", e)
	}
	if !strings.Contains(rendered, "SECOND ASSIGNMENT") || !strings.Contains(rendered, "[truncated") {
		t.Fatal("ASSERT_ASSIGNMENT_REPLAY: lost independent or fragmented launch")
	}
	if _, e = execute(append(base, "--payload", "--json", "--snapshot", first.Snapshot)...); e == nil {
		t.Fatal("ASSERT_ASSIGNMENT_BINDING: assignment option mismatch accepted")
	}
}

func TestReviewAssignmentParityMissingAndDefault(t *testing.T) {
	path := handoffFixture(t)
	review, e := buildReviewTails(path, "archive", "canonical", "", 2, "", 2, false, 1, "", true)
	if e != nil {
		t.Fatal(e)
	}
	context, e := buildTranscriptContext(path, "AAA", 2, 1, "")
	if e != nil {
		t.Fatal(e)
	}
	launches := func(p ledgerPage, id string) map[string]string {
		out := map[string]string{}
		for _, raw := range p.Events {
			var event map[string]json.RawMessage
			if e := json.Unmarshal(raw, &event); e != nil {
				t.Fatal(e)
			}
			if ledgerString(event, "agent_id") == id && ledgerString(event, "section") == "launch" {
				out[ledgerString(event, "event_id")] = string(raw)
			}
		}
		return out
	}
	actual, want := launches(review, "AAA"), launches(context, "AAA")
	if len(actual) != 2 || len(want) != 2 {
		t.Fatal("ASSERT_ASSIGNMENT_PARITY: independent launches lost")
	}
	for id, raw := range want {
		if actual[id] != raw {
			t.Fatal("ASSERT_ASSIGNMENT_PARITY: assignment identity/body differs from context")
		}
	}
	_, execute := setupNotebook(t)
	text, e := execute("transcript", "review", path, "--queue", "archive", "--limit", "2", "--last", "2", "--include-assignment", "--format", "text")
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(text, "[assignment unavailable]") || !strings.Contains(text, "join: missing") {
		t.Fatalf("ASSERT_ASSIGNMENT_MISSING: missing join not qualified: %s", text)
	}
	plain, e := buildReviewTails(path, "archive", "canonical", "", 2, "", 2, true, 1, "")
	if e != nil {
		t.Fatal(e)
	}
	for _, raw := range plain.Events {
		var event map[string]json.RawMessage
		_ = json.Unmarshal(raw, &event)
		if ledgerString(event, "section") == "launch" || event["include_assignment"] != nil || event["selected_assignments"] != nil || event["assignment_events"] != nil {
			t.Fatal("ASSERT_ASSIGNMENT_OPT_IN: default review changed")
		}
	}
}

func TestAssignmentQuestionRoutesInitialRetrieval(t *testing.T) {
	_, execute := setupNotebook(t)
	for owner, terms := range map[string][]string{
		"context":     {"initial", "single-room", "initial inspection envelope"},
		"review":      {"--include-assignment", "selected_assignments", "recent events only"},
		"rooms":       {"For assignment alignment", "nn transcript context"},
		"interaction": {"not a rigid", "necessary bounded evidence directly", "not simply because another read follows"},
	} {
		text, e := execute("skills", "get", "nn-transcript", "--reference", owner)
		if e != nil {
			t.Fatal(e)
		}
		for _, term := range terms {
			if !strings.Contains(text, term) {
				t.Errorf("ASSERT_INITIAL_ASSIGNMENT_ROUTING: %s missing %q", owner, term)
			}
		}
	}
}
