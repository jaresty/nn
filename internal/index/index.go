// Package index manages the SQLite cache of note metadata.
// The index is derived from the Markdown files and can be fully rebuilt at any time.
package index

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/jaresty/nn/internal/note"
)

const schema = `
CREATE TABLE IF NOT EXISTS notes (
    id           TEXT PRIMARY KEY,
    title        TEXT NOT NULL,
    type         TEXT NOT NULL,
    status       TEXT NOT NULL,
    tags         TEXT NOT NULL DEFAULT '',
    created      TEXT NOT NULL,
    modified     TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    path         TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS links (
    from_id    TEXT NOT NULL,
    to_id      TEXT NOT NULL,
    annotation TEXT NOT NULL,
    PRIMARY KEY (from_id, to_id)
);

CREATE TABLE IF NOT EXISTS tags (
    note_id TEXT NOT NULL,
    tag     TEXT NOT NULL,
    PRIMARY KEY (note_id, tag)
);
`

// Index wraps an open SQLite database.
type Index struct {
	db *sql.DB
}

// Open opens (or creates) the index database at dbPath and ensures the schema exists.
func Open(dbPath string) (*Index, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("index.Open: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("index.Open: schema: %w", err)
	}
	return &Index{db: db}, nil
}

// Close releases the database connection.
func (idx *Index) Close() error {
	return idx.db.Close()
}

// TableExists returns nil if the named table exists, or an error otherwise.
func (idx *Index) TableExists(name string) error {
	var n int
	err := idx.db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, name,
	).Scan(&n)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("table %q does not exist", name)
	}
	return nil
}

// Rebuild clears the index and repopulates it by scanning all .md files in dir.
func (idx *Index) Rebuild(dir string) error {
	tx, err := idx.db.Begin()
	if err != nil {
		return fmt.Errorf("index.Rebuild: begin: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	for _, tbl := range []string{"notes", "links", "tags"} {
		if _, err := tx.Exec("DELETE FROM " + tbl); err != nil {
			return fmt.Errorf("index.Rebuild: clear %s: %w", tbl, err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("index.Rebuild: readdir: %w", err)
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("index.Rebuild: read %s: %w", e.Name(), err)
		}
		n, err := note.Parse(data)
		if err != nil {
			continue // skip non-note .md files
		}
		hash := contentHash(data)
		tagsStr := strings.Join(n.Tags, ",")

		_, err = tx.Exec(`
			INSERT OR REPLACE INTO notes (id, title, type, status, tags, created, modified, content_hash, path)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			n.ID, n.Title, string(n.Type), string(n.Status),
			tagsStr,
			n.Created.UTC().Format("2006-01-02T15:04:05Z"),
			n.Modified.UTC().Format("2006-01-02T15:04:05Z"),
			hash, path,
		)
		if err != nil {
			return fmt.Errorf("index.Rebuild: insert note %s: %w", n.ID, err)
		}

		for _, t := range n.Tags {
			if _, err := tx.Exec(`INSERT OR REPLACE INTO tags (note_id, tag) VALUES (?, ?)`,
				n.ID, t); err != nil {
				return fmt.Errorf("index.Rebuild: insert tag: %w", err)
			}
		}

		for _, lnk := range n.Links {
			if _, err := tx.Exec(`INSERT OR REPLACE INTO links (from_id, to_id, annotation) VALUES (?, ?, ?)`,
				n.ID, lnk.TargetID, lnk.Annotation); err != nil {
				return fmt.Errorf("index.Rebuild: insert link: %w", err)
			}
		}
	}

	return tx.Commit()
}

// CountNotes returns the number of rows in the notes table.
func (idx *Index) CountNotes() (int, error) {
	var n int
	if err := idx.db.QueryRow("SELECT COUNT(*) FROM notes").Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// CountLinks returns the number of rows in the links table.
func (idx *Index) CountLinks() (int, error) {
	var n int
	if err := idx.db.QueryRow("SELECT COUNT(*) FROM links").Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// IsStale returns true if the file at path has changed since it was indexed.
// Stale detection uses content hash (sha256) of the file.
func (idx *Index) IsStale(id, path string) (bool, error) {
	var storedHash string
	err := idx.db.QueryRow("SELECT content_hash FROM notes WHERE id = ?", id).Scan(&storedHash)
	if err == sql.ErrNoRows {
		return true, nil // not indexed at all → stale
	}
	if err != nil {
		return false, fmt.Errorf("index.IsStale: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("index.IsStale: read: %w", err)
	}
	return contentHash(data) != storedHash, nil
}

// contentHash returns the hex-encoded SHA-256 of data.
func contentHash(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}
