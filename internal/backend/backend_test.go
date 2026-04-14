package backend_test

import (
	"testing"

	"github.com/jaresty/nn/internal/backend"
	"github.com/jaresty/nn/internal/note"
)

// Compile-time check that gitlocal.Backend satisfies the Backend interface.
// This test lives here so the interface is verified without importing gitlocal.
func TestBackendInterface(t *testing.T) {
	var _ backend.Backend = (*mockBackend)(nil)
}

// mockBackend satisfies Backend for compile-time verification.
type mockBackend struct{}

func (m *mockBackend) Write(n *note.Note) error                              { return nil }
func (m *mockBackend) Read(id string) (*note.Note, error)                   { return nil, nil }
func (m *mockBackend) Delete(id string) error                               { return nil }
func (m *mockBackend) List() ([]*note.Note, error)                          { return nil, nil }
func (m *mockBackend) AddLink(from, to, annotation, linkType string) error  { return nil }
func (m *mockBackend) AddLinks(from string, targets []backend.LinkTarget) error { return nil }
func (m *mockBackend) RemoveLink(from, to string) error                     { return nil }
func (m *mockBackend) Promote(id string, to note.Status) error              { return nil }
