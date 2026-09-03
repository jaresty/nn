package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	im "github.com/jaresty/nn/internal/media"
)

type processRunnerFunc func(context.Context, im.CommandSpec) (im.CommandResult, error)

func (f processRunnerFunc) Run(ctx context.Context, spec im.CommandSpec) (im.CommandResult, error) {
	return f(ctx, spec)
}

func TestProductionMediaServiceInspectPersistsAcrossInstances(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "..", "internal", "media", "testdata", "ffprobe-video-audio.json"))
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "movie.mp4")
	if err := os.WriteFile(source, []byte("tiny fixture identity"), 0o600); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	runner := processRunnerFunc(func(_ context.Context, spec im.CommandSpec) (im.CommandResult, error) {
		if spec.Executable != "ffprobe" {
			t.Fatalf("unexpected executable %q", spec.Executable)
		}
		return im.CommandResult{Started: true, Stdout: fixture}, nil
	})
	first := newProductionMediaServiceAt(root, runner)
	result, err := first.Execute(context.Background(), mediaCommandRequest{Operation: "inspect", Source: source, NonInteractive: true})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != "succeeded" || result.RunID == "" || strings.Contains(result.ErrorsString(), "media_service_unavailable") {
		t.Fatalf("production inspect result: %#v", result)
	}
	fresh := newProductionMediaServiceAt(root, runner)
	discovered, err := fresh.Discover(context.Background(), result.RunID)
	if err != nil {
		t.Fatal(err)
	}
	packetAny, err := fresh.Context(context.Background(), result.RunID, 32768, 1)
	if err != nil {
		t.Fatal(err)
	}
	packet := packetAny.(im.ContextPacket)
	if discovered.RunID != result.RunID || packet.SourceMetadata.VideoCodec != "h264" {
		t.Fatalf("fresh production service recovery: %#v %#v", discovered, packet)
	}
}

func TestProductionMediaServicePersistsDocumentLevelTranscriptEvidence(t *testing.T) {
	source := filepath.Join(t.TempDir(), "input.mov")
	if err := os.WriteFile(source, []byte("identity"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := processRunnerFunc(func(_ context.Context, spec im.CommandSpec) (im.CommandResult, error) {
		if spec.Executable == "ffmpeg" {
			_ = os.WriteFile(spec.Args[len(spec.Args)-1], []byte("wav"), 0o600)
			return im.CommandResult{}, nil
		}
		return im.CommandResult{Stdout: []byte(`{"duration":353.5166875,"inference_time":10.787305333,"text":"hello"}`)}, nil
	})
	service := newProductionMediaServiceAt(t.TempDir(), runner).(*productionMediaService)
	result, err := service.Execute(context.Background(), mediaCommandRequest{Operation: "transcribe", Source: source, Engine: "parakeet"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != "succeeded" {
		t.Fatalf("non-empty document transcript succeeds: %#v", result)
	}
	stored, err := service.store.Discover(context.Background(), result.RunID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, evidence := range stored.Manifest.Evidence {
		if evidence.Kind == im.EvidenceToolReport {
			found = true
		}
	}
	if !found {
		t.Fatalf("transcript tool evidence persisted: %#v", stored.Manifest.Evidence)
	}
}

func TestProductionMediaServiceSamplingFlagsAndPrepareSharePipeline(t *testing.T) {
	fixture, err := os.ReadFile(filepath.Join("..", "..", "..", "internal", "media", "testdata", "ffprobe-video-audio.json"))
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "movie.mp4")
	if err := os.WriteFile(source, []byte("identity"), 0o600); err != nil {
		t.Fatal(err)
	}
	var commands []im.CommandSpec
	runner := processRunnerFunc(func(_ context.Context, spec im.CommandSpec) (im.CommandResult, error) {
		commands = append(commands, spec)
		switch spec.Executable {
		case "ffprobe":
			return im.CommandResult{Started: true, Stdout: fixture}, nil
		case "ffmpeg":
			if err := os.WriteFile(spec.Args[len(spec.Args)-1], []byte("artifact"), 0o600); err != nil {
				return im.CommandResult{}, err
			}
			return im.CommandResult{Started: true}, nil
		case "parakeet":
			data, _ := json.Marshal(map[string]any{"duration": 1.0, "text": "prepared transcript"})
			return im.CommandResult{Started: true, Stdout: data}, nil
		default:
			t.Fatalf("unexpected executable %q", spec.Executable)
			return im.CommandResult{}, nil
		}
	})
	service := newProductionMediaServiceAt(t.TempDir(), runner)
	result, err := service.Execute(context.Background(), mediaCommandRequest{Operation: "sample", Source: source, Every: "200s", Frames: []string{"00:01:00"}, ContactSheet: true})
	if err != nil {
		t.Fatal(err)
	}
	joined := commandArgs(commands)
	for _, required := range []string{"00:00:00.000", "00:01:00.000", "00:03:20.000", "tile=4x1"} {
		if !strings.Contains(joined, required) {
			t.Errorf("sample flags affect execution: missing %q in %s", required, joined)
		}
	}
	stored, err := service.(*productionMediaService).store.Discover(context.Background(), result.RunID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Manifest.Evidence) < 5 {
		t.Fatalf("sample artifacts persist as manifest evidence: %#v", stored.Manifest.Evidence)
	}

	commands = nil
	capture, err := service.Execute(context.Background(), mediaCommandRequest{Operation: "prepare", Source: source, Every: "200s", Frames: []string{"00:01:00"}, ContactSheet: true, Engine: "parakeet"})
	if err != nil {
		t.Fatal(err)
	}
	if got := stageNames(capture.Stages); got != "inspect,sample,transcribe" {
		t.Fatalf("prepare combines requested pipeline stages: %s", got)
	}
	joined = commandArgs(commands)
	if !strings.Contains(joined, "00:01:00.000") || !strings.Contains(joined, "tile=4x1") || !strings.Contains(joined, "parakeet") {
		t.Fatalf("prepare executes sampling and explicit transcription: %s", joined)
	}
	before := len(commands)
	fresh := newProductionMediaServiceAt(service.(*productionMediaService).root, runner)
	packetAny, err := fresh.Context(context.Background(), capture.RunID, 5, 1)
	if err != nil {
		t.Fatal(err)
	}
	packet := packetAny.(im.ContextPacket)
	if len(commands) != before {
		t.Fatalf("context retrieval does no reprocessing: before=%d after=%d", before, len(commands))
	}
	if packet.SchemaVersion != im.ContextPacketSchemaVersion || len(packet.Transcript) == 0 || !packet.Truncated || packet.NextPage != 2 {
		t.Fatalf("bounded packet discloses pagination: %#v", packet)
	}
	if len(packet.Images) == 0 {
		t.Fatal("context packet includes image attachments")
	}
	for _, image := range packet.Images {
		if image.MIME != "image/jpeg" {
			t.Fatalf("typed image MIME: %#v", image)
		}
		if _, err := os.ReadFile(image.Path); err != nil {
			t.Fatalf("image attachment is readable: %v", err)
		}
	}
}

func commandArgs(commands []im.CommandSpec) string {
	var parts []string
	for _, c := range commands {
		parts = append(parts, c.Executable+" "+strings.Join(c.Args, " "))
	}
	return strings.Join(parts, "\n")
}
func stageNames(stages []mediaStageResult) string {
	var names []string
	for _, s := range stages {
		names = append(names, s.Name)
	}
	return strings.Join(names, ",")
}

func (r mediaCommandResult) ErrorsString() string {
	var b strings.Builder
	for _, e := range r.Errors {
		b.WriteString(e.Code)
		b.WriteString(e.Message)
	}
	return b.String()
}
