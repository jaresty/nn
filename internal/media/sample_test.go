package media

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestBuildSamplingPlanDeterministic(t *testing.T) {
	plan, err := BuildSamplingPlan(5*time.Second, 2*time.Second, []time.Duration{3 * time.Second, time.Second, 3 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Duration{0, time.Second, 2 * time.Second, 3 * time.Second, 4 * time.Second}
	if !reflect.DeepEqual(plan.Timestamps, want) {
		t.Fatalf("sampling timestamps mismatch: got %v want %v", plan.Timestamps, want)
	}
	if got := FormatTimestamp(3723400 * time.Millisecond); got != "01:02:03.400" {
		t.Fatalf("timestamp format mismatch: got %q", got)
	}
}

func TestBuildSamplingPlanExcludesDurationEndpoint(t *testing.T) {
	plan, err := BuildSamplingPlan(4*time.Second, 2*time.Second, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Duration{0, 2 * time.Second}
	if !reflect.DeepEqual(plan.Timestamps, want) {
		t.Fatalf("sampling duration endpoint included: got %v want %v", plan.Timestamps, want)
	}
}

func TestBuildSamplingPlanRejectsInvalidValues(t *testing.T) {
	if _, err := BuildSamplingPlan(time.Second, 0, nil); err == nil {
		t.Fatal("zero sampling interval accepted")
	}
	if _, err := BuildSamplingPlan(time.Second, time.Second, []time.Duration{-1}); err == nil {
		t.Fatal("negative explicit timestamp accepted")
	}
}

func TestExtractFramesUsesArgumentVectorsAndValidatesOutputs(t *testing.T) {
	dir := t.TempDir()
	var commands []CommandSpec
	runner := runnerFunc(func(_ context.Context, command CommandSpec) (CommandResult, error) {
		commands = append(commands, command)
		return CommandResult{}, os.WriteFile(command.Args[len(command.Args)-1], []byte("frame"), 0600)
	})
	artifacts, err := ExtractFrames(context.Background(), runner, "/tmp/a movie.mp4", []time.Duration{time.Second}, dir)
	if err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(dir, "frame-00-00-01.000.jpg")
	if len(artifacts) != 1 || artifacts[0].Timestamp != time.Second || artifacts[0].Path != wantPath {
		t.Fatalf("frame artifacts mismatch: %#v", artifacts)
	}
	want := CommandSpec{Executable: "ffmpeg", Args: []string{"-v", "error", "-ss", "00:00:01.000", "-i", "/tmp/a movie.mp4", "-frames:v", "1", "-y", wantPath}, Streams: StreamPolicy{CaptureStderr: true}}
	if !reflect.DeepEqual(commands, []CommandSpec{want}) {
		t.Fatalf("frame argv mismatch: got %#v want %#v", commands, []CommandSpec{want})
	}
}

func TestExtractFramesRejectsMissingOutput(t *testing.T) {
	runner := runnerFunc(func(context.Context, CommandSpec) (CommandResult, error) { return CommandResult{}, nil })
	if _, err := ExtractFrames(context.Background(), runner, "video.mp4", []time.Duration{0}, t.TempDir()); err == nil {
		t.Fatal("missing frame output accepted")
	}
}

func TestCreateContactSheetUsesFilterAndValidatesOutput(t *testing.T) {
	dir := t.TempDir()
	var observed CommandSpec
	runner := runnerFunc(func(_ context.Context, command CommandSpec) (CommandResult, error) {
		observed = command
		return CommandResult{}, os.WriteFile(command.Args[len(command.Args)-1], []byte("sheet"), 0600)
	})
	path := filepath.Join(dir, "contact.jpg")
	if err := CreateContactSheet(context.Background(), runner, "video.mp4", 353*time.Second, 45*time.Second, 4, path); err != nil {
		t.Fatal(err)
	}
	filter := strings.Join(observed.Args, " ")
	if strings.Contains(filter, "x0") {
		t.Fatalf("contact sheet layout rejects zero dimension: %s", filter)
	}
	if !strings.Contains(filter, "fps=1/45,tile=4x2") {
		t.Fatalf("contact sheet layout is deterministic for duration/sample count: %s", filter)
	}
}
