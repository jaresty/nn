package note_test

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestRequiresLinkType(t *testing.T) {
	if !note.IsKnownLinkType("requires") {
		t.Error("requires should be a known link type")
	}
}

func TestIsDone(t *testing.T) {
	t.Run("no checkboxes is done vacuously", func(t *testing.T) {
		if !note.IsDone("no checkboxes here") {
			t.Error("expected IsDone=true for note with no checkboxes")
		}
	})
	t.Run("all checked is done", func(t *testing.T) {
		body := "- [x] first\n- [x] second\n"
		if !note.IsDone(body) {
			t.Error("expected IsDone=true when all boxes checked")
		}
	})
	t.Run("one unchecked is not done", func(t *testing.T) {
		body := "- [x] first\n- [ ] second\n"
		if note.IsDone(body) {
			t.Error("expected IsDone=false when unchecked box present")
		}
	})
	t.Run("only unchecked is not done", func(t *testing.T) {
		body := "- [ ] only item\n"
		if note.IsDone(body) {
			t.Error("expected IsDone=false for unchecked box")
		}
	})
	t.Run("inline mention of - [ ] in prose is not a checkbox", func(t *testing.T) {
		body := "a note is done when all its `- [ ]` items are checked\n"
		if !note.IsDone(body) {
			t.Error("expected IsDone=true — inline backtick mention is not a real checkbox")
		}
	})
	t.Run("inline mention of - [ ] in prose does not affect HasCheckbox", func(t *testing.T) {
		body := "use `- [ ]` syntax to create checkboxes\n"
		if note.HasCheckbox(body) {
			t.Error("expected HasCheckbox=false — inline mention is not a real checkbox")
		}
	})
}
