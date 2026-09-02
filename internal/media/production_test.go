package media

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestOSProcessRunnerBoundsOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture")
	}
	r := OSProcessRunner{}
	got, err := r.Run(context.Background(), CommandSpec{Executable: "sh", Args: []string{"-c", "printf 123456789; printf abcdefghi >&2"}, Streams: StreamPolicy{CaptureStdout: true, CaptureStderr: true}, LogLimitBytes: 5})
	if err != nil {
		t.Fatal(err)
	}
	if string(got.Stdout) != "12345" || string(got.Stderr) != "abcde" {
		t.Fatalf("runner bounds stdout/stderr: stdout=%q stderr=%q", got.Stdout, got.Stderr)
	}
}

func TestOSProcessRunnerCancellation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	got, err := (OSProcessRunner{}).Run(ctx, CommandSpec{Executable: "sh", Args: []string{"-c", "sleep 30"}})
	if !errors.Is(err, context.Canceled) || !got.Cancelled {
		t.Fatalf("runner cancellation terminates process: result=%#v err=%v", got, err)
	}
}

func TestLocalRunStoreSurvivesFreshInstance(t *testing.T) {
	root := t.TempDir()
	first := NewLocalRunStore(root)
	run := StoredRun{Result: RunResult{SchemaVersion: RunResultSchemaVersion, Command: "capture", RunID: "run-1", BundleID: "bundle-1", Outcome: OutcomePartial}, Manifest: Manifest{SchemaVersion: ManifestSchemaVersion, RunID: "run-1", BundleID: "bundle-1", Lifecycle: LifecycleCompleted, Publication: PublicationPublished}, Projection: NoteProjection{Title: "Media run-1", Markdown: "## Coverage\nPartial visual evidence."}}
	if err := first.Publish(context.Background(), run); err != nil {
		t.Fatal(err)
	}
	fresh := NewLocalRunStore(root)
	got, err := fresh.Discover(context.Background(), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	projection, err := fresh.Project(context.Background(), "run-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.Result.RunID != "run-1" || !strings.Contains(projection.Markdown, "## Coverage") {
		t.Fatalf("fresh service discovers and projects persisted run: %#v %#v", got, projection)
	}
}

func TestOSProcessRunnerDeadline(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture")
	}
	got, err := (OSProcessRunner{}).Run(context.Background(), CommandSpec{Executable: "sh", Args: []string{"-c", "sleep 30"}, Deadline: time.Millisecond})
	if err == nil || !got.TimedOut {
		t.Fatalf("runner deadline terminates process: result=%#v err=%v", got, err)
	}
}
