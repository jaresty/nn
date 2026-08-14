package rules

import (
	"testing"
	"time"

	"github.com/jaresty/nn/internal/note"
)

func hasFact(facts []Fact, key string) bool {
	for _, f := range facts {
		if f.Key() == key {
			return true
		}
	}
	return false
}

func TestFactsFromNote(t *testing.T) {
	exp := time.Date(2026, 8, 21, 0, 0, 0, 0, time.UTC)
	n := &note.Note{
		ID:             "n1",
		Type:           note.TypeModel,
		Status:         note.StatusReviewed,
		Tags:           []string{"daily", "x"},
		Representation: "taxonomy",
		Expires:        &exp,
		Body:           "prose\n- [ ] do a thing\n- [x] already done\n- [ ] another",
		Links: []note.Link{
			{TargetID: "n2", Type: "refines"},
			{TargetID: "n3", Type: "governs"},
		},
	}

	facts := FactsFromNotes([]*note.Note{n})

	for _, want := range []string{
		`note(n1,model,reviewed)`,
		`link(n1,n2,refines)`,
		`link(n1,n3,governs)`,
		`tag(n1,daily)`,
		`tag(n1,x)`,
		`representation(n1,taxonomy)`,
		`open_item(n1,do a thing)`,
		`open_item(n1,another)`,
	} {
		if !hasFact(facts, want) {
			t.Errorf("missing fact %s", want)
		}
	}

	// A checked item must NOT produce an open_item fact.
	if hasFact(facts, `open_item(n1,already done)`) {
		t.Error("checked item leaked into open_item facts")
	}
	// expires fact carries the date (YYYY-MM-DD).
	if !hasFact(facts, `expires(n1,2026-08-21)`) {
		t.Error("missing expires fact")
	}
}

func TestFactsFromNote_NoRepresentationOrExpires(t *testing.T) {
	n := &note.Note{ID: "n9", Type: note.TypeConcept, Status: note.StatusDraft}
	facts := FactsFromNotes([]*note.Note{n})
	for _, f := range facts {
		if f.Pred == "representation" || f.Pred == "expires" {
			t.Errorf("unexpected optional fact: %s", f.Key())
		}
	}
	if !hasFact(facts, `note(n9,concept,draft)`) {
		t.Error("missing base note fact")
	}
}
