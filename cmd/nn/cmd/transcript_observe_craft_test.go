package cmd

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestTranscriptObserveCraft(t *testing.T) {
	path := reviewFixture(t)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.WriteString("\n" + `{"type":"custom","customType":"subagents:record","data":{"id":"00-empty","type":"worker","status":"completed"}}` + "\n")
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	_, execute := setupNotebook(t)
	observed, err := execute("transcript", "observe", path)
	if err != nil {
		t.Fatal(err)
	}
	sections := strings.Split(observed, "\n## ")[1:]
	t.Run("sample", func(t *testing.T) {
		t.Log("Procedure: compare emitted stream IDs with ROOT plus native canonical direct-child page limited to two")
		t.Log("Assertion: CRAFT_SAMPLE")
		raw, err := execute("transcript", "tree", path, "--parent", "ROOT", "--limit", "2", "--json")
		if err != nil {
			t.Fatal(err)
		}
		var page treeChildPage
		if err = json.Unmarshal([]byte(raw), &page); err != nil {
			t.Fatal(err)
		}
		if page.TotalChildren < 3 {
			t.Fatal("fixture must distinguish bounded from exhaustive selection")
		}
		want := []string{"ROOT"}
		for _, c := range page.Children {
			want = append(want, c.ID)
		}
		got := []string{}
		for _, s := range sections {
			got = append(got, strings.SplitN(s, " — ", 2)[0])
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("CRAFT_SAMPLE: FAIL got=%v want=%v", got, want)
		}
		t.Log("CRAFT_SAMPLE: PASS")
	})
	t.Run("tails", func(t *testing.T) {
		t.Log("Procedure: compare each emitted stream body byte-for-byte with native events --last 5 --format text --max-text-chars 1000")
		t.Log("Assertion: CRAFT_TAILS")
		for _, s := range sections {
			parts := strings.SplitN(s, "\n", 2)
			id := strings.SplitN(parts[0], " — ", 2)[0]
			want, err := execute("transcript", "events", path, id, "--last", "5", "--format", "text", "--max-text-chars", "1000")
			if err != nil {
				t.Fatal(err)
			}
			if len(parts) != 2 || parts[1] != want {
				t.Fatalf("CRAFT_TAILS: FAIL stream=%s", id)
			}
		}
		t.Log("CRAFT_TAILS: PASS")
	})
	t.Run("dispatch", func(t *testing.T) {
		t.Log("Procedure: inspect served initial recipe commands and forbidden eager-loading directives; publication only, not model behavior")
		t.Log("Assertion: CRAFT_DISPATCH")
		body, err := execute("skills", "get", "nn-transcript", "--reference", "observe")
		if err != nil {
			t.Fatal(err)
		}
		commands := []string{}
		for _, line := range strings.Split(body, "\n") {
			if strings.HasPrefix(line, "nn transcript ") {
				commands = append(commands, strings.Fields(line)[2])
			}
		}
		if !reflect.DeepEqual(commands, []string{"ls", "observe"}) {
			t.Fatal("CRAFT_DISPATCH: FAIL recipe")
		}
		for _, owner := range []string{"interaction", "navigate", "events"} {
			if strings.Contains(body, "Load `nn skills get nn-transcript --reference "+owner+"` first") {
				t.Fatal("CRAFT_DISPATCH: FAIL eager owner")
			}
		}
		t.Log("CRAFT_DISPATCH: PASS")
	})
}
