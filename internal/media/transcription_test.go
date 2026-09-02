package media

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestTranscriptResultPreservesConformanceMetadata(t *testing.T) {
	confidence := 0.91
	got := TranscriptResult{Segments: []TranscriptSegment{{Start: time.Second, End: 2 * time.Second, Text: "hello", Confidence: &confidence}}, Engine: EngineIdentity{Name: "parakeet", Version: "0.1"}, Language: "en", EffectiveConfig: map[string]string{"model": "tdt", "token": "[REDACTED]"}}
	if len(got.Segments) != 1 || got.EffectiveConfig["token"] != "[REDACTED]" {
		t.Fatalf("metadata not preserved: %#v", got)
	}
}
func TestParakeetDeclaresExperimentalCapabilities(t *testing.T) {
	got := NewParakeetAdapter("parakeet", nil).Capabilities()
	want := TranscriptionCapabilities{InputModes: []InputMode{InputPreparedAudio}, TimestampOrigin: TimestampMediaStart, Confidence: ConfidenceOptional, RequiresModel: true, UsesCache: true, Trust: TrustExperimentalLocal}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("capabilities mismatch\n got: %#v\nwant: %#v", got, want)
	}
}

func TestParakeetPreparesMOVInvokesCLIAndNormalizesJSON(t *testing.T) {
	runner := &recordingCommandRunner{run: func(spec CommandSpec) (CommandResult, error) {
		if spec.Executable == "ffmpeg" {
			if err := os.WriteFile(spec.Args[len(spec.Args)-1], []byte("wav"), 0o600); err != nil {
				return CommandResult{}, err
			}
			return CommandResult{}, nil
		}
		return CommandResult{Stdout: []byte(`{"text":"hello world","language":"en","sentences":[{"start":0.25,"end":1.5,"text":"hello world"}]}`)}, nil
	}}
	got, err := NewParakeetAdapter("/opt/parakeet", runner).Transcribe(context.Background(), TranscriptionRequest{InputPath: "/tmp/input.mov"})
	if err != nil {
		t.Fatal(err)
	}
	if len(runner.specs) != 2 {
		t.Fatalf("prepare and transcribe command count: %d", len(runner.specs))
	}
	prepared := runner.specs[0].Args[len(runner.specs[0].Args)-1]
	wantPrepare := CommandSpec{Executable: "ffmpeg", Args: []string{"-v", "error", "-i", "/tmp/input.mov", "-vn", "-ac", "1", "-ar", "16000", "-c:a", "pcm_s16le", "-y", prepared}, Streams: StreamPolicy{CaptureStderr: true}}
	wantTranscribe := CommandSpec{Executable: "/opt/parakeet", Args: []string{"transcribe", prepared, "--format", "json"}, Streams: StreamPolicy{CaptureStdout: true, CaptureStderr: true}}
	if !reflect.DeepEqual(runner.specs[0], wantPrepare) || !reflect.DeepEqual(runner.specs[1], wantTranscribe) {
		t.Fatalf("exact argv mismatch:\n%#v\n%#v", runner.specs[0], runner.specs[1])
	}
	if len(got.Segments) != 1 || got.Segments[0].Start != 250*time.Millisecond || got.Segments[0].End != 1500*time.Millisecond || got.Segments[0].Text != "hello world" || got.Language != "en" {
		t.Fatalf("actual JSON normalization: %#v", got)
	}
	if _, err := os.Stat(prepared); !os.IsNotExist(err) {
		t.Fatalf("prepared WAV cleaned after success: %v", err)
	}
}

func TestParakeetFailureAndCancellationCleanPreparedWAV(t *testing.T) {
	for _, tc := range []struct {
		name    string
		failure error
	}{{"failure", errors.New("parakeet failed")}, {"cancellation", context.Canceled}} {
		t.Run(tc.name, func(t *testing.T) {
			var prepared string
			runner := &recordingCommandRunner{run: func(spec CommandSpec) (CommandResult, error) {
				if spec.Executable == "ffmpeg" {
					prepared = spec.Args[len(spec.Args)-1]
					_ = os.WriteFile(prepared, []byte("wav"), 0o600)
					return CommandResult{}, nil
				}
				return CommandResult{}, tc.failure
			}}
			_, err := NewParakeetAdapter("parakeet", runner).Transcribe(context.Background(), TranscriptionRequest{InputPath: "input.mov"})
			if !errors.Is(err, tc.failure) {
				t.Fatalf("failure preserved: %v", err)
			}
			if prepared == "" {
				t.Fatal("MOV was not prepared")
			}
			if _, statErr := os.Stat(prepared); !os.IsNotExist(statErr) {
				t.Fatalf("prepared WAV cleaned: %v", statErr)
			}
		})
	}
}
func TestParakeetNormalizesLucatacoDocumentJSON(t *testing.T) {
	payload := `{"duration":353.5166875,"inference_time":10.787305333,"text":" observed transcript "}`
	runner := parakeetPayloadRunner(payload)
	got, err := NewParakeetAdapter("parakeet", runner).Transcribe(context.Background(), TranscriptionRequest{InputPath: "input.mov"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Segments) != 1 || got.Segments[0].Start != 0 || got.Segments[0].End != 353516687500*time.Nanosecond || got.Segments[0].Text != "observed transcript" {
		t.Fatalf("document JSON normalizes to bounded segment: %#v", got)
	}
	if got.EffectiveConfig["timestamps"] != "synthesized-document-boundaries" {
		t.Fatalf("document timing provenance explicit: %#v", got.EffectiveConfig)
	}
}

func TestParakeetRejectsPayloadsWithNoTranscriptSegments(t *testing.T) {
	for _, payload := range []string{`{"duration":353.5166875,"inference_time":10.7,"text":"   "}`, `{"duration":3,"unknown":"value"}`, `{}`} {
		_, err := NewParakeetAdapter("parakeet", parakeetPayloadRunner(payload)).Transcribe(context.Background(), TranscriptionRequest{InputPath: "input.mov"})
		if err == nil || !strings.Contains(err.Error(), "no transcript segments") {
			t.Errorf("zero-segment payload rejected: payload=%s err=%v", payload, err)
		}
	}
}

func parakeetPayloadRunner(payload string) *recordingCommandRunner {
	return &recordingCommandRunner{run: func(spec CommandSpec) (CommandResult, error) {
		if spec.Executable == "ffmpeg" {
			_ = os.WriteFile(spec.Args[len(spec.Args)-1], []byte("wav"), 0o600)
			return CommandResult{}, nil
		}
		return CommandResult{Stdout: []byte(payload)}, nil
	}}
}

func TestParakeetMalformedOutputIsExplicit(t *testing.T) {
	runner := &recordingCommandRunner{run: func(spec CommandSpec) (CommandResult, error) {
		if spec.Executable == "ffmpeg" {
			_ = os.WriteFile(spec.Args[len(spec.Args)-1], []byte("wav"), 0o600)
			return CommandResult{}, nil
		}
		return CommandResult{Stdout: []byte("not-json")}, nil
	}}
	_, err := NewParakeetAdapter("parakeet", runner).Transcribe(context.Background(), TranscriptionRequest{InputPath: "audio.mov"})
	if err == nil || !strings.Contains(err.Error(), "decode parakeet output") {
		t.Fatalf("malformed output error: %v", err)
	}
}

type recordingCommandRunner struct {
	specs []CommandSpec
	run   func(CommandSpec) (CommandResult, error)
}

func (r *recordingCommandRunner) Run(_ context.Context, spec CommandSpec) (CommandResult, error) {
	r.specs = append(r.specs, spec)
	return r.run(spec)
}
