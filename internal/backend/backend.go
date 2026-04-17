// Package backend defines the Backend interface for note storage.
package backend

import "github.com/jaresty/nn/internal/note"

// LinkTarget is a (toID, annotation, optional type) pair used by AddLinks.
type LinkTarget struct {
	ToID       string
	Annotation string
	Type       string // optional
}

// LinkUpdate is a (toID, optional annotation, optional type) used by BulkUpdateLinks.
type LinkUpdate struct {
	ToID       string
	Annotation *string // nil = leave unchanged
	Type       *string // nil = leave unchanged
}

// Backend abstracts note storage so the CLI can be tested and extended
// without depending on a specific implementation.
type Backend interface {
	Write(n *note.Note) error
	Read(id string) (*note.Note, error)
	Delete(id string) error
	List() ([]*note.Note, error)
	AddLink(fromID, toID, annotation, linkType string) error
	AddLinks(fromID string, targets []LinkTarget) error
	RemoveLink(fromID, toID string) error
	Promote(id string, to note.Status) error
	Update(n *note.Note) error
	UpdateLink(fromID, toID string, annotation, linkType *string) error
	BulkUpdateLinks(fromID string, updates []LinkUpdate) error
	BulkWrite(notes []*note.Note) error
}
