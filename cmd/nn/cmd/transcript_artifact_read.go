package cmd

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const transcriptArtifactReadMaxBytes = 16384

var (
	errArtifactInvalid     = errors.New("artifact read: invalid request")
	errArtifactStale       = errors.New("artifact read: stale snapshot")
	errArtifactMissing     = errors.New("artifact read: missing")
	errArtifactDenied      = errors.New("artifact read: denied")
	errArtifactNonregular  = errors.New("artifact read: nonregular")
	errArtifactUnsupported = errors.New("artifact read: unsupported platform")
	errArtifactRead        = errors.New("artifact read: read failure")
)

type artifactReadFile interface {
	io.Reader
	Stat() (os.FileInfo, error)
	Close() error
}

type transcriptArtifactReadResult struct {
	Version       string `json:"version"`
	EventID       string `json:"event_id"`
	Snapshot      string `json:"diagnostic_snapshot"`
	FindingIndex  int    `json:"finding_index"`
	Location      string `json:"location"`
	Kind          string `json:"kind"`
	RecordedPath  string `json:"recorded_path"`
	AllowRoot     string `json:"allow_root"`
	AcquiredBytes int    `json:"acquired_bytes"`
	ContentBase64 string `json:"content_base64"`
	SHA256        string `json:"sha256"`
	Coverage      string `json:"coverage"`
	MeasuredSize  *int64 `json:"measured_size,omitempty"`
	Qualification string `json:"qualification"`
}

func newTranscriptArtifactCmd() *cobra.Command {
	c := &cobra.Command{Use: "artifact", Short: "Read explicitly selected transcript artifacts"}
	c.AddCommand(newTranscriptArtifactReadCmd())
	return c
}

func newTranscriptArtifactReadCmd() *cobra.Command {
	return newTranscriptArtifactReadCmdUsing(ledgerRecords, openTranscriptArtifact)
}

func newTranscriptArtifactReadCmdUsing(acquire func(string, string) ([]ledgerRecord, string, string, error), opener func(string, string) (artifactReadFile, error)) *cobra.Command {
	var eventID, snapshot, root string
	var finding int
	c := &cobra.Command{Use: "read <session> <agent-id>", Short: "Read one explicitly admitted recorded transcript artifact", Args: cobra.ExactArgs(2), RunE: func(c *cobra.Command, args []string) error {
		for _, name := range []string{"event", "snapshot", "finding", "allow-root"} {
			if !c.Flags().Changed(name) {
				return errArtifactInvalid
			}
		}
		if eventID == "" || finding < 1 || !validArtifactPath(root) {
			return errArtifactInvalid
		}
		records, schema, detail, err := acquire(args[0], args[1])
		if err != nil {
			return fmt.Errorf("%w", errArtifactMissing)
		}
		selects := []string{"identity", "message", "usage", "tools", "lifecycle"}
		events, err := projectLedger(records, args[1], selects, true)
		if err != nil {
			return fmt.Errorf("%w", errArtifactMissing)
		}
		page, err := buildLedgerPageUsing(args[0], args[1], schema, detail, selects, false, events, 1, "", eventID, false, nil, nil, true)
		if err != nil {
			return fmt.Errorf("%w", errArtifactMissing)
		}
		if page.Snapshot != snapshot {
			return errArtifactStale
		}
		var selected transcriptArtifactDiagnostic
		found := false
		for _, event := range events {
			if stringValue(event["event_id"]) == eventID {
				selected = diagnoseTranscriptArtifactEvent(event)
				found = true
				break
			}
		}
		if !found || selected.EventID != eventID || finding > len(selected.Findings) {
			return fmt.Errorf("%w", errArtifactMissing)
		}
		ref := selected.Findings[finding-1].Artifact
		if ref == nil || ref.RecordedPath == "" {
			return fmt.Errorf("%w", errArtifactMissing)
		}
		path := ref.RecordedPath
		if !validArtifactPath(path) || !artifactWithinRoot(root, path) {
			return errArtifactDenied
		}
		f, err := opener(root, path)
		if err != nil {
			return classifyArtifactOpenError(err)
		}
		defer f.Close()
		before, err := f.Stat()
		if err != nil {
			return errArtifactRead
		}
		if !before.Mode().IsRegular() {
			return errArtifactNonregular
		}
		b, err := io.ReadAll(io.LimitReader(f, transcriptArtifactReadMaxBytes+1))
		if err != nil {
			return errArtifactRead
		}
		after, err := f.Stat()
		if err != nil {
			return errArtifactRead
		}
		measured := before.Size()
		if measured < 0 {
			measured = 0
		}
		n := len(b)
		coverage := "complete"
		if n > transcriptArtifactReadMaxBytes {
			b = b[:transcriptArtifactReadMaxBytes]
			n = len(b)
			coverage = "partial"
		}
		if !sameArtifactMetadata(before, after) {
			coverage = "unstable"
		}
		sum := sha256.Sum256(b)
		result := transcriptArtifactReadResult{Version: "nn.transcript.artifact-read/v1", EventID: eventID, Snapshot: snapshot, FindingIndex: finding, Location: selected.Findings[finding-1].Location, Kind: selected.Findings[finding-1].Kind, RecordedPath: path, AllowRoot: root, AcquiredBytes: n, ContentBase64: base64.StdEncoding.EncodeToString(b), SHA256: hex.EncodeToString(sum[:]), Coverage: coverage, MeasuredSize: &measured, Qualification: "New acquisition of present bytes only; does not establish historical artifact identity."}
		enc, err := json.Marshal(result)
		if err != nil {
			return errArtifactRead
		}
		_, err = c.OutOrStdout().Write(append(enc, '\n'))
		return err
	}}
	c.Flags().StringVar(&eventID, "event", "", "exact recorded event ID")
	c.Flags().StringVar(&snapshot, "snapshot", "", "exact-event default diagnostic snapshot")
	c.Flags().IntVar(&finding, "finding", 0, "one-based diagnostic finding")
	c.Flags().StringVar(&root, "allow-root", "", "absolute approved artifact directory")
	return c
}

func validArtifactPath(p string) bool {
	if p == "" || strings.IndexByte(p, 0) >= 0 || !filepath.IsAbs(p) || filepath.Clean(p) != p || strings.Contains(p, "://") {
		return false
	}
	for _, part := range strings.Split(p, string(filepath.Separator)) {
		if part == "." || part == ".." {
			return false
		}
	}
	return true
}
func artifactWithinRoot(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
func sameArtifactMetadata(a, b os.FileInfo) bool {
	return a.Size() == b.Size() && a.Mode() == b.Mode() && a.ModTime() == b.ModTime()
}
func classifyArtifactOpenError(err error) error {
	if errors.Is(err, errArtifactUnsupported) {
		return errArtifactUnsupported
	}
	if errors.Is(err, os.ErrNotExist) {
		return errArtifactMissing
	}
	if errors.Is(err, os.ErrPermission) {
		return errArtifactDenied
	}
	return errArtifactDenied
}
