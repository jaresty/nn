package media

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"
)

type runnerFunc func(context.Context, CommandSpec) (CommandResult, error)

func (f runnerFunc) Run(ctx context.Context, command CommandSpec) (CommandResult, error) {
	return f(ctx, command)
}

func TestParseProbeMetadata(t *testing.T) {
	data, err := os.ReadFile("testdata/ffprobe-video-audio.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseProbeMetadata(data)
	if err != nil {
		t.Fatal(err)
	}
	want := Metadata{Duration: 353240 * time.Millisecond, Formats: []string{"mov", "mp4", "m4a", "3gp", "3g2", "mj2"}, VideoCodec: "h264", Width: 1920, Height: 1080, FrameRate: Rational{Numerator: 30000, Denominator: 1001}, HasAudio: true}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized metadata mismatch: got %#v want %#v", got, want)
	}
}

func TestParseProbeMetadataVideoOnly(t *testing.T) {
	data, err := os.ReadFile("testdata/ffprobe-video-only.json")
	if err != nil {
		t.Fatal(err)
	}
	got, err := ParseProbeMetadata(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.HasAudio {
		t.Fatal("audio presence mismatch: got true want false")
	}
}

func TestInspectInvokesFFprobeWithoutShell(t *testing.T) {
	fixture, err := os.ReadFile("testdata/ffprobe-video-audio.json")
	if err != nil {
		t.Fatal(err)
	}
	var observed CommandSpec
	runner := runnerFunc(func(_ context.Context, command CommandSpec) (CommandResult, error) {
		observed = command
		return CommandResult{Stdout: fixture}, nil
	})
	_, err = Inspect(context.Background(), runner, "/tmp/a movie.mp4")
	if err != nil {
		t.Fatal(err)
	}
	want := CommandSpec{Executable: "ffprobe", Args: []string{"-v", "error", "-show_streams", "-show_format", "-of", "json", "/tmp/a movie.mp4"}, Streams: StreamPolicy{CaptureStdout: true, CaptureStderr: true}}
	if !reflect.DeepEqual(observed, want) {
		t.Fatalf("ffprobe argv mismatch: got %#v want %#v", observed, want)
	}
}

func TestInspectRejectsMalformedOutput(t *testing.T) {
	runner := runnerFunc(func(context.Context, CommandSpec) (CommandResult, error) {
		return CommandResult{Stdout: []byte("not-json")}, nil
	})
	if _, err := Inspect(context.Background(), runner, "bad.mp4"); err == nil {
		t.Fatal("malformed probe output accepted")
	}
}

func TestInspectPropagatesRunnerFailure(t *testing.T) {
	want := errors.New("runner failed")
	runner := runnerFunc(func(context.Context, CommandSpec) (CommandResult, error) { return CommandResult{}, want })
	if _, err := Inspect(context.Background(), runner, "bad.mp4"); !errors.Is(err, want) {
		t.Fatalf("runner error mismatch: got %v want %v", err, want)
	}
}
