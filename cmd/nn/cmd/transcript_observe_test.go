package cmd

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestTranscriptObserveNativeComposition(t *testing.T) {
	for _, fixture := range []struct {
		name string
		make func(*testing.T) string
	}{
		{"pi", reviewFixture}, {"claude", claudeCodeFixture}, {"sdk", func(t *testing.T) string { return writeSDKCLIFixture(t, t.TempDir()) }},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			path := fixture.make(t)
			_, execute := setupNotebook(t)
			text, err := execute("transcript", "observe", path)
			if err != nil {
				t.Fatal(err)
			}
			tree, err := execute("transcript", "tree", path, "--parent", "ROOT", "--limit", "2", "--json")
			if err != nil {
				t.Fatal(err)
			}
			var p treeChildPage
			if err = json.Unmarshal([]byte(tree), &p); err != nil {
				t.Fatal(err)
			}
			ids := []string{"ROOT"}
			for _, child := range p.Children {
				ids = append(ids, child.ID)
			}
			if len(ids) > 3 || strings.Count(text, "\n## ") != len(ids) {
				t.Fatal("unbounded or incomplete sample", text)
			}
			for _, id := range ids {
				tail, err := execute("transcript", "events", path, id, "--last", "5", "--format", "text", "--max-text-chars", "1000")
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(text, tail) {
					t.Fatal("native tail/identity/availability not preserved", id)
				}
			}
			for _, s := range []string{"omitted direct children:", "not recency or importance", "not an atomic capture", "older events remain uninspected"} {
				if !strings.Contains(text, s) {
					t.Fatal("missing qualification", s)
				}
			}
		})
	}
}

func TestTranscriptObserveErrorPublishesNoPartialRead(t *testing.T) {
	c := newTranscriptObserveCmd()
	var out bytes.Buffer
	c.SetOut(&out)
	if err := c.RunE(c, []string{t.TempDir() + "/absent.jsonl"}); err == nil {
		t.Fatal("missing source accepted")
	}
	if out.Len() != 0 {
		t.Fatal("partial observation published", out.String())
	}
}

func TestTranscriptObserveCompactDispatch(t *testing.T) {
	_, execute := setupNotebook(t)
	body, err := execute("skills", "get", "nn-transcript", "--reference", "observe")
	if err != nil {
		t.Fatal(err)
	}
	if len(strings.Split(body, "\n")) > 75 || len(body) > 5000 {
		t.Fatal("initial owner grew beyond compact budget")
	}
	if !strings.Contains(body, "initial bounded read") || !strings.Contains(body, "nn transcript observe <session-id>") {
		t.Fatal("missing self-contained entry point")
	}
	for _, old := range []string{"Load `nn skills get nn-transcript --reference interaction` first", "nn transcript events <session>", "nn transcript tree <session>"} {
		if strings.Contains(body, old) {
			t.Fatal("eager fan-out restored", old)
		}
	}
}
