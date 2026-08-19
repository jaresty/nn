package cmd

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/backend"
	"github.com/jaresty/nn/internal/note"
)

// stubBackend is a minimal backend that records Write calls.
type stubBackend struct {
	written []*note.Note
}

func (s *stubBackend) Write(n *note.Note) error                                           { s.written = append(s.written, n); return nil }
func (s *stubBackend) List() ([]*note.Note, error)                                        { return nil, nil }
func (s *stubBackend) ListMeta() ([]*note.Note, error)                                    { return nil, nil }
func (s *stubBackend) Read(id string) (*note.Note, error)                                 { return nil, nil }
func (s *stubBackend) Delete(id string) error                                             { return nil }
func (s *stubBackend) BulkWrite(notes []*note.Note) error                                 { return nil }
func (s *stubBackend) AddLink(fromID, toID, annotation, linkType, linkStatus string) error { return nil }
func (s *stubBackend) AddLinks(fromID string, targets []backend.LinkTarget) error          { return nil }
func (s *stubBackend) RemoveLink(fromID, toID string) error                               { return nil }
func (s *stubBackend) RemoveLinkByType(fromID, toID, linkType string) error               { return nil }
func (s *stubBackend) Promote(id string, from time.Time, to note.Status) error            { return nil }
func (s *stubBackend) Update(n *note.Note, since *time.Time) error                        { return nil }
func (s *stubBackend) UpdateLink(fromID, toID string, annotation, linkType, linkStatus *string) error {
	return nil
}
func (s *stubBackend) BulkUpdateLinks(fromID string, updates []backend.LinkUpdate) error { return nil }
func (s *stubBackend) BulkApply(newNotes []*note.Note, updateNotes []*note.Note) error   { return nil }

func TestFetchCaptureCreatesNote(t *testing.T) {
	// property [2b]: --capture creates a note whose body contains fetched plaintext
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>Capture Test</title></head><body><p>Important content here.</p></body></html>`))
	}))
	defer srv.Close()

	stub := &stubBackend{}
	state := &rootState{backend: stub, notebookDir: ""}

	var stdout, stderr bytes.Buffer
	err := runFetch(srv.URL, true, &stdout, &stderr, state)
	if err != nil {
		t.Fatalf("runFetch --capture: %v", err)
	}

	if len(stub.written) == 0 {
		t.Fatal("property [2b]: --capture did not create any note")
	}
	n := stub.written[0]
	if !strings.Contains(n.Body, "Important content here") {
		t.Errorf("property [2b]: captured note body %q does not contain fetched content", n.Body)
	}
	if n.Type != note.TypeObservation {
		t.Errorf("property [2b]: expected type observation, got %q", n.Type)
	}
}
