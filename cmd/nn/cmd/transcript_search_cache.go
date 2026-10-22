package cmd

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const transcriptSearchCacheVersion = 2

const transcriptSearchCacheSchema = `
CREATE TABLE IF NOT EXISTS transcript_search_meta (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS transcript_search_files (
    path TEXT PRIMARY KEY,
    size INTEGER NOT NULL,
    mtime_ns INTEGER NOT NULL,
    session_id TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS transcript_search_events (
    path TEXT NOT NULL,
    ordinal INTEGER NOT NULL,
    agent_id TEXT NOT NULL,
    event_id TEXT NOT NULL,
    timestamp TEXT NOT NULL,
    role TEXT NOT NULL,
    meaningful TEXT NOT NULL,
    normalized TEXT NOT NULL,
    message_digest BLOB NOT NULL,
    PRIMARY KEY (path, ordinal),
    FOREIGN KEY (path) REFERENCES transcript_search_files(path) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS transcript_search_events_path ON transcript_search_events(path, ordinal);
`

type transcriptSearchCache struct {
	db *sql.DB
}

type transcriptSearchCacheStats struct {
	RefreshedFiles int
	DecodedRecords int
}

type transcriptSearchCachedEvent struct {
	ordinal    int
	agentID    string
	eventID    string
	timestamp  string
	role       string
	meaningful string
	normalized string
	digest     [32]byte
}

func openTranscriptSearchCache(dbPath string) (*transcriptSearchCache, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("transcript search cache: mkdir: %w", err)
	}
	dsn := dbPath + "?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("transcript search cache: open: %w", err)
	}
	db.SetMaxOpenConns(1)
	cache := &transcriptSearchCache{db: db}
	if err := cache.ensureSchema(); err != nil {
		db.Close()
		return nil, err
	}
	return cache, nil
}

func (c *transcriptSearchCache) Close() error { return c.db.Close() }

func (c *transcriptSearchCache) ensureSchema() error {
	if _, err := c.db.Exec(`CREATE TABLE IF NOT EXISTS transcript_search_meta (key TEXT PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("transcript search cache: metadata schema: %w", err)
	}
	var version int
	err := c.db.QueryRow(`SELECT CAST(value AS INTEGER) FROM transcript_search_meta WHERE key = 'schema_version'`).Scan(&version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("transcript search cache: read schema version: %w", err)
	}
	if err == nil && version != transcriptSearchCacheVersion {
		if _, err := c.db.Exec(`
DROP TABLE IF EXISTS transcript_search_events;
DROP TABLE IF EXISTS transcript_search_files;
DELETE FROM transcript_search_meta WHERE key = 'schema_version';`); err != nil {
			return fmt.Errorf("transcript search cache: reset schema: %w", err)
		}
	}
	if _, err := c.db.Exec(transcriptSearchCacheSchema); err != nil {
		return fmt.Errorf("transcript search cache: schema: %w", err)
	}
	if _, err := c.db.Exec(`INSERT INTO transcript_search_meta(key, value) VALUES('schema_version', ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`, transcriptSearchCacheVersion); err != nil {
		return fmt.Errorf("transcript search cache: store schema version: %w", err)
	}
	return nil
}

func canonicalTranscriptPath(path string) (string, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

func (c *transcriptSearchCache) Search(files []string, match func(string) bool, agentFilter, before string, raw bool, limit int) (transcriptSearchResult, transcriptSearchCacheStats, error) {
	stats, canonical, err := c.refresh(files)
	if err != nil {
		return transcriptSearchResult{}, stats, err
	}
	result := transcriptSearchResult{Matches: []transcriptSearchMatch{}}
	for i, path := range canonical {
		sessionID, events, err := c.events(path)
		if err != nil {
			return transcriptSearchResult{}, stats, err
		}
		for _, event := range events {
			if agentFilter != "" && event.agentID != agentFilter {
				continue
			}
			if before != "" && event.timestamp != "" && event.timestamp >= before {
				continue
			}
			if raw {
				return transcriptSearchResult{}, stats, fmt.Errorf("transcript search cache: raw search requires source scan")
			}
			text := event.meaningful
			if !match(text) {
				continue
			}
			if len(result.Matches) == limit {
				result.Truncated = true
				continue
			}
			result.Matches = append(result.Matches, transcriptSearchMatch{
				recordOrdinal: event.ordinal,
				messageDigest: event.digest,
				Session:       sessionID,
				AgentID:       event.agentID,
				EventID:       event.eventID,
				Timestamp:     event.timestamp,
				Role:          event.role,
				Excerpt:       text,
				SourcePath:    files[i],
			})
		}
	}
	result.Returned = len(result.Matches)
	return result, stats, nil
}

func (c *transcriptSearchCache) SearchSubstring(files []string, query, agentFilter, before string, limit int) (transcriptSearchResult, transcriptSearchCacheStats, error) {
	stats, canonical, err := c.refresh(files)
	if err != nil {
		return transcriptSearchResult{}, stats, err
	}
	tx, err := c.db.Begin()
	if err != nil {
		return transcriptSearchResult{}, stats, fmt.Errorf("transcript search cache: begin query: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err = tx.Exec(`CREATE TEMP TABLE IF NOT EXISTS transcript_search_selected(path TEXT PRIMARY KEY, position INTEGER NOT NULL, source_path TEXT NOT NULL); DELETE FROM transcript_search_selected;`); err != nil {
		return transcriptSearchResult{}, stats, fmt.Errorf("transcript search cache: prepare selected paths: %w", err)
	}
	for i, path := range canonical {
		if _, err = tx.Exec(`INSERT INTO transcript_search_selected(path, position, source_path) VALUES(?, ?, ?) ON CONFLICT(path) DO UPDATE SET position=excluded.position, source_path=excluded.source_path`, path, i, files[i]); err != nil {
			return transcriptSearchResult{}, stats, fmt.Errorf("transcript search cache: select path: %w", err)
		}
	}
	rows, err := tx.Query(`
SELECT f.session_id, e.ordinal, e.agent_id, e.event_id, e.timestamp, e.role, e.meaningful, e.message_digest, s.source_path
FROM transcript_search_selected s
JOIN transcript_search_files f ON f.path = s.path
JOIN transcript_search_events e ON e.path = s.path
WHERE instr(e.normalized, ?) > 0
  AND (? = '' OR e.agent_id = ?)
  AND (? = '' OR e.timestamp = '' OR e.timestamp < ?)
ORDER BY s.position, e.ordinal
LIMIT ?`, strings.ToLower(query), agentFilter, agentFilter, before, before, limit+1)
	if err != nil {
		return transcriptSearchResult{}, stats, fmt.Errorf("transcript search cache: query: %w", err)
	}
	defer rows.Close()
	result := transcriptSearchResult{Matches: []transcriptSearchMatch{}}
	for rows.Next() {
		var match transcriptSearchMatch
		var digest []byte
		if err := rows.Scan(&match.Session, &match.recordOrdinal, &match.AgentID, &match.EventID, &match.Timestamp, &match.Role, &match.Excerpt, &digest, &match.SourcePath); err != nil {
			return transcriptSearchResult{}, stats, fmt.Errorf("transcript search cache: scan match: %w", err)
		}
		if len(digest) != len(match.messageDigest) {
			return transcriptSearchResult{}, stats, fmt.Errorf("transcript search cache: invalid message digest length %d", len(digest))
		}
		copy(match.messageDigest[:], digest)
		if len(result.Matches) == limit {
			result.Truncated = true
			continue
		}
		result.Matches = append(result.Matches, match)
	}
	if err := rows.Err(); err != nil {
		return transcriptSearchResult{}, stats, fmt.Errorf("transcript search cache: iterate matches: %w", err)
	}
	result.Returned = len(result.Matches)
	return result, stats, nil
}

func (c *transcriptSearchCache) refresh(files []string) (transcriptSearchCacheStats, []string, error) {
	var stats transcriptSearchCacheStats
	if err := c.pruneDeleted(); err != nil {
		return stats, nil, err
	}
	canonical := make([]string, len(files))
	for i, path := range files {
		resolved, err := canonicalTranscriptPath(path)
		if err != nil {
			return stats, nil, fmt.Errorf("search %s: %w", path, err)
		}
		canonical[i] = resolved
		info, err := os.Stat(resolved)
		if err != nil {
			return stats, nil, fmt.Errorf("search %s: %w", path, err)
		}
		var size, mtime int64
		err = c.db.QueryRow(`SELECT size, mtime_ns FROM transcript_search_files WHERE path = ?`, resolved).Scan(&size, &mtime)
		if err == nil && size == info.Size() && mtime == info.ModTime().UnixNano() {
			continue
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return stats, nil, fmt.Errorf("transcript search cache: inspect %s: %w", path, err)
		}
		sessionID, events, decoded, err := materializeTranscriptSearchFile(resolved)
		stats.DecodedRecords += decoded
		if err != nil {
			return stats, nil, fmt.Errorf("search %s: %w", path, err)
		}
		if err := c.replaceFile(resolved, info.Size(), info.ModTime().UnixNano(), sessionID, events); err != nil {
			return stats, nil, err
		}
		stats.RefreshedFiles++
	}
	return stats, canonical, nil
}

func materializeTranscriptSearchFile(path string) (string, []transcriptSearchCachedEvent, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", nil, 0, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	sessionID := filepath.Base(path)
	sessionID = sessionID[:len(sessionID)-len(filepath.Ext(sessionID))]
	headerFound := false
	ordinal, decoded := 0, 0
	var events []transcriptSearchCachedEvent
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var record rawRecord
		if json.Unmarshal(line, &record) != nil {
			continue
		}
		decoded++
		ordinal++
		normalizeCodexRecord(&record)
		if !headerFound && record.Type == "session" && record.ID != "" {
			sessionID, headerFound = record.ID, true
		}
		if !isPiEventRecord(record) || len(record.Message) == 0 {
			continue
		}
		var message struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		}
		if json.Unmarshal(record.Message, &message) != nil {
			continue
		}
		agentID := record.AgentID
		if agentID == "" {
			agentID = "ROOT"
		}
		role := message.Role
		if role == "" {
			role = record.Type
		}
		eventID := record.ID
		if eventID == "" {
			eventID = fmt.Sprintf("record:%d", ordinal)
		}
		meaningful := stringsTrimSpaceMeaningful(message.Role, message.Content)
		events = append(events, transcriptSearchCachedEvent{
			ordinal:    ordinal,
			agentID:    agentID,
			eventID:    eventID,
			timestamp:  record.Timestamp,
			role:       role,
			meaningful: meaningful,
			normalized: strings.ToLower(meaningful),
			digest:     sha256.Sum256(record.Message),
		})
	}
	if err := scanner.Err(); err != nil {
		return "", nil, decoded, err
	}
	return sessionID, events, decoded, nil
}

func stringsTrimSpaceMeaningful(role string, content json.RawMessage) string {
	return string(bytes.TrimSpace([]byte(meaningfulContent(role, content))))
}

func (c *transcriptSearchCache) replaceFile(path string, size, mtime int64, sessionID string, events []transcriptSearchCachedEvent) error {
	tx, err := c.db.Begin()
	if err != nil {
		return fmt.Errorf("transcript search cache: begin refresh: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck
	if _, err := tx.Exec(`DELETE FROM transcript_search_events WHERE path = ?`, path); err != nil {
		return fmt.Errorf("transcript search cache: clear events: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO transcript_search_files(path, size, mtime_ns, session_id) VALUES(?, ?, ?, ?) ON CONFLICT(path) DO UPDATE SET size=excluded.size, mtime_ns=excluded.mtime_ns, session_id=excluded.session_id`, path, size, mtime, sessionID); err != nil {
		return fmt.Errorf("transcript search cache: store file: %w", err)
	}
	for _, event := range events {
		if _, err := tx.Exec(`INSERT INTO transcript_search_events(path, ordinal, agent_id, event_id, timestamp, role, meaningful, normalized, message_digest) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)`, path, event.ordinal, event.agentID, event.eventID, event.timestamp, event.role, event.meaningful, event.normalized, event.digest[:]); err != nil {
			return fmt.Errorf("transcript search cache: store event: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("transcript search cache: commit refresh: %w", err)
	}
	return nil
}

func (c *transcriptSearchCache) events(path string) (string, []transcriptSearchCachedEvent, error) {
	var sessionID string
	if err := c.db.QueryRow(`SELECT session_id FROM transcript_search_files WHERE path = ?`, path).Scan(&sessionID); err != nil {
		return "", nil, fmt.Errorf("transcript search cache: read file: %w", err)
	}
	rows, err := c.db.Query(`SELECT ordinal, agent_id, event_id, timestamp, role, meaningful, normalized, message_digest FROM transcript_search_events WHERE path = ? ORDER BY ordinal`, path)
	if err != nil {
		return "", nil, fmt.Errorf("transcript search cache: read events: %w", err)
	}
	defer rows.Close()
	var events []transcriptSearchCachedEvent
	for rows.Next() {
		var event transcriptSearchCachedEvent
		var digest []byte
		if err := rows.Scan(&event.ordinal, &event.agentID, &event.eventID, &event.timestamp, &event.role, &event.meaningful, &event.normalized, &digest); err != nil {
			return "", nil, fmt.Errorf("transcript search cache: scan event: %w", err)
		}
		if len(digest) != len(event.digest) {
			return "", nil, fmt.Errorf("transcript search cache: invalid message digest length %d", len(digest))
		}
		copy(event.digest[:], digest)
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return "", nil, fmt.Errorf("transcript search cache: iterate events: %w", err)
	}
	return sessionID, events, nil
}

func (c *transcriptSearchCache) pruneDeleted() error {
	rows, err := c.db.Query(`SELECT path FROM transcript_search_files`)
	if err != nil {
		return fmt.Errorf("transcript search cache: list files: %w", err)
	}
	var deleted []string
	for rows.Next() {
		var path string
		if err := rows.Scan(&path); err != nil {
			rows.Close()
			return fmt.Errorf("transcript search cache: scan file: %w", err)
		}
		if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
			deleted = append(deleted, path)
		} else if err != nil {
			rows.Close()
			return fmt.Errorf("transcript search cache: stat %s: %w", path, err)
		}
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("transcript search cache: close file list: %w", err)
	}
	for _, path := range deleted {
		if _, err := c.db.Exec(`DELETE FROM transcript_search_files WHERE path = ?`, path); err != nil {
			return fmt.Errorf("transcript search cache: prune %s: %w", path, err)
		}
	}
	return nil
}
