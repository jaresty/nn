package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"syscall"
	"unicode/utf8"
)

type OSProcessRunner struct{}

type boundedBuffer struct {
	buf       bytes.Buffer
	remaining int64
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	n := len(p)
	if b.remaining > 0 {
		keep := int64(len(p))
		if keep > b.remaining {
			keep = b.remaining
		}
		_, _ = b.buf.Write(p[:keep])
		b.remaining -= keep
	}
	return n, nil
}
func (b *boundedBuffer) Bytes() []byte { return append([]byte(nil), b.buf.Bytes()...) }

func (OSProcessRunner) Run(ctx context.Context, spec CommandSpec) (CommandResult, error) {
	if spec.Executable == "" {
		return CommandResult{}, fmt.Errorf("media runner: executable is required")
	}
	if spec.Deadline > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, spec.Deadline)
		defer cancel()
	}
	cmd := exec.CommandContext(ctx, spec.Executable, spec.Args...)
	cmd.Dir = spec.WorkingDirectory
	if spec.Environment != nil {
		keys := make([]string, 0, len(spec.Environment))
		for k := range spec.Environment {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		cmd.Env = make([]string, 0, len(keys))
		for _, k := range keys {
			cmd.Env = append(cmd.Env, k+"="+spec.Environment[k])
		}
	}
	limit := spec.LogLimitBytes
	if limit <= 0 {
		limit = 1 << 20
	}
	stdout := &boundedBuffer{remaining: limit}
	stderr := &boundedBuffer{remaining: limit}
	if spec.Streams.CaptureStdout {
		cmd.Stdout = stdout
	} else {
		cmd.Stdout = io.Discard
	}
	if spec.Streams.CaptureStderr {
		cmd.Stderr = stderr
	} else {
		cmd.Stderr = io.Discard
	}
	result := CommandResult{}
	err := cmd.Start()
	if err != nil {
		if ctx.Err() != nil {
			result.Cancelled = errors.Is(ctx.Err(), context.Canceled)
			result.TimedOut = errors.Is(ctx.Err(), context.DeadlineExceeded)
			return result, ctx.Err()
		}
		return result, fmt.Errorf("media runner start: %w", err)
	}
	result.Started = true
	err = cmd.Wait()
	result.Stdout = stdout.Bytes()
	result.Stderr = stderr.Bytes()
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
		if status, ok := cmd.ProcessState.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			result.Signal = status.Signal().String()
		}
	}
	if ctx.Err() != nil {
		result.Cancelled = errors.Is(ctx.Err(), context.Canceled)
		result.TimedOut = errors.Is(ctx.Err(), context.DeadlineExceeded)
		return result, ctx.Err()
	}
	if err != nil {
		return result, fmt.Errorf("media runner exit: %w", err)
	}
	return result, nil
}

const ContextPacketSchemaVersion = "nn.media.context/v1"

type ImageAttachment struct {
	Path      string          `json:"path"`
	MIME      string          `json:"mime"`
	Timestamp *MediaTimestamp `json:"timestamp,omitempty"`
	Role      string          `json:"role"`
}
type TranscriptChunk struct {
	Segments []TranscriptSegment `json:"segments"`
	Text     string              `json:"text"`
}
type ContextPacket struct {
	SchemaVersion   string            `json:"schema_version"`
	RunID           RunID             `json:"run_id"`
	SourceID        SourceID          `json:"source_id"`
	BundleID        BundleID          `json:"bundle_id"`
	ManifestLocator string            `json:"manifest_locator"`
	Coverage        Coverage          `json:"coverage"`
	Transcript      []TranscriptChunk `json:"transcript_chunks"`
	Images          []ImageAttachment `json:"image_attachments"`
	Page            int               `json:"page"`
	Truncated       bool              `json:"truncated"`
	NextPage        int               `json:"next_page,omitempty"`
}
type NoteProjection struct {
	Title    string `json:"title"`
	Markdown string `json:"markdown"`
}
type StoredRun struct {
	Result     RunResult           `json:"result"`
	Manifest   Manifest            `json:"manifest"`
	Projection NoteProjection      `json:"projection"`
	Transcript []TranscriptSegment `json:"transcript,omitempty"`
	Images     []ImageAttachment   `json:"images,omitempty"`
}
type LocalRunStore struct{ root string }

func NewLocalRunStore(root string) *LocalRunStore { return &LocalRunStore{root: root} }
func (s *LocalRunStore) runDir(id string) string  { return filepath.Join(s.root, "runs", id) }
func (s *LocalRunStore) Publish(_ context.Context, run StoredRun) error {
	id := string(run.Result.RunID)
	if id == "" {
		id = string(run.Manifest.RunID)
	}
	if id == "" {
		return fmt.Errorf("publish run: run id is required")
	}
	if run.Manifest.RunID == "" {
		run.Manifest.RunID = RunID(id)
	}
	if run.Result.RunID == "" {
		run.Result.RunID = RunID(id)
	}
	if err := run.Manifest.Validate(); err != nil {
		return fmt.Errorf("publish run: %w", err)
	}
	parent := filepath.Join(s.root, "runs")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	staging, err := os.MkdirTemp(parent, ".staging-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(staging)
	manifestPath := filepath.Join(staging, "manifest.json")
	run.Result.ManifestLocator = filepath.Join(s.runDir(id), "manifest.json")
	if err := writeJSONSync(manifestPath, run.Manifest); err != nil {
		return err
	}
	if err := writeJSONSync(filepath.Join(staging, "run.json"), run); err != nil {
		return err
	}
	if err := syncDir(staging); err != nil {
		return err
	}
	if err := os.Rename(staging, s.runDir(id)); err != nil {
		return fmt.Errorf("publish run atomically: %w", err)
	}
	return syncDir(parent)
}
func writeJSONSync(path string, v any) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	encErr := json.NewEncoder(f).Encode(v)
	syncErr := f.Sync()
	closeErr := f.Close()
	if encErr != nil {
		return encErr
	}
	if syncErr != nil {
		return syncErr
	}
	return closeErr
}
func syncDir(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}
func (s *LocalRunStore) Discover(_ context.Context, id string) (StoredRun, error) {
	data, err := os.ReadFile(filepath.Join(s.runDir(id), "run.json"))
	if err != nil {
		return StoredRun{}, fmt.Errorf("discover run %q: %w", id, err)
	}
	var run StoredRun
	if err := json.Unmarshal(data, &run); err != nil {
		return StoredRun{}, fmt.Errorf("decode run %q: %w", id, err)
	}
	return run, nil
}
func (s *LocalRunStore) Project(ctx context.Context, id string) (NoteProjection, error) {
	run, err := s.Discover(ctx, id)
	if err != nil {
		return NoteProjection{}, err
	}
	if run.Projection.Markdown == "" {
		return NoteProjection{}, fmt.Errorf("run %q has no note projection", id)
	}
	return run.Projection, nil
}
func (s *LocalRunStore) Context(ctx context.Context, id string, maxBytes, page int) (ContextPacket, error) {
	run, err := s.Discover(ctx, id)
	if err != nil {
		return ContextPacket{}, err
	}
	if maxBytes <= 0 {
		maxBytes = 32768
	}
	if page < 1 {
		page = 1
	}
	pages := [][]TranscriptChunk{{}}
	used := 0
	for _, segment := range run.Transcript {
		remaining := segment.Text
		for remaining != "" {
			available := maxBytes - used
			if available == 0 {
				pages = append(pages, nil)
				used = 0
				available = maxBytes
			}
			n := len(remaining)
			if n > available {
				n = available
				for n > 0 && !utf8.RuneStart(remaining[n]) {
					n--
				}
			}
			if n == 0 {
				return ContextPacket{}, fmt.Errorf("context max-bytes %d is smaller than one UTF-8 rune", maxBytes)
			}
			part := remaining[:n]
			copySegment := segment
			copySegment.Text = part
			pages[len(pages)-1] = append(pages[len(pages)-1], TranscriptChunk{Segments: []TranscriptSegment{copySegment}, Text: part})
			used += n
			remaining = remaining[n:]
		}
	}
	packet := ContextPacket{SchemaVersion: ContextPacketSchemaVersion, RunID: run.Result.RunID, SourceID: run.Result.SourceID, BundleID: run.Result.BundleID, ManifestLocator: run.Result.ManifestLocator, Coverage: run.Manifest.Coverage, Images: run.Images, Page: page}
	if page <= len(pages) {
		packet.Transcript = pages[page-1]
	}
	if page < len(pages) {
		packet.Truncated = true
		packet.NextPage = page + 1
	}
	return packet, nil
}
