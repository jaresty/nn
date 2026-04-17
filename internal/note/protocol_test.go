package note_test

import (
	"testing"

	"github.com/jaresty/nn/internal/note"
)

func TestTypeProtocolIsValid(t *testing.T) {
	if !note.TypeProtocol.IsValid() {
		t.Errorf("TypeProtocol.IsValid() = false, want true")
	}
}

func TestTypeProtocolNotMisspelled(t *testing.T) {
	if string(note.TypeProtocol) != "protocol" {
		t.Errorf("TypeProtocol = %q, want %q", note.TypeProtocol, "protocol")
	}
}
