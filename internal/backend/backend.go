// Package backend defines the Backend interface for note storage.
package backend

import "github.com/jaresty/nn/internal/note"

// Backend abstracts note storage so the CLI can be tested and extended
// without depending on a specific implementation.
type Backend interface {
	Write(n *note.Note) error
	Read(id string) (*note.Note, error)
	Delete(id string) error
	List() ([]*note.Note, error)
	AddLink(fromID, toID, annotation string) error
	RemoveLink(fromID, toID string) error
	Promote(id string, to note.Status) error
}
