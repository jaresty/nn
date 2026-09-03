package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jaresty/nn/internal/backend"
	"github.com/jaresty/nn/internal/note"
)

type fakeMediaService struct {
	executeCalls, discoverCalls, doctorCalls int
	request                                  mediaCommandRequest
	result                                   mediaCommandResult
}

func (f *fakeMediaService) Execute(_ context.Context, r mediaCommandRequest) (mediaCommandResult, error) {
	f.executeCalls++
	f.request = r
	return f.result, nil
}
func (f *fakeMediaService) Discover(_ context.Context, id string) (mediaCommandResult, error) {
	f.discoverCalls++
	f.request.RunID = id
	return f.result, nil
}
func (f *fakeMediaService) Context(_ context.Context, id string, _, _ int) (any, error) {
	f.discoverCalls++
	f.request.RunID = id
	return map[string]any{"schema_version": "nn.media.context/v1", "run_id": id}, nil
}
func (f *fakeMediaService) Doctor(_ context.Context, r mediaCommandRequest) (mediaCommandResult, error) {
	f.doctorCalls++
	f.request = r
	return f.result, nil
}

type recordingBackend struct {
	writes  int
	written *note.Note
}

func (b *recordingBackend) Write(n *note.Note) error                                 { b.writes++; b.written = n; return nil }
func (*recordingBackend) Read(string) (*note.Note, error)                            { return nil, nil }
func (*recordingBackend) Delete(string) error                                        { return nil }
func (*recordingBackend) List() ([]*note.Note, error)                                { return nil, nil }
func (*recordingBackend) AddLink(string, string, string, string, string) error       { return nil }
func (*recordingBackend) AddLinks(string, []backend.LinkTarget) error                { return nil }
func (*recordingBackend) SetLinkType(string, string, string, string) error           { return nil }
func (*recordingBackend) RemoveLink(string, string) error                            { return nil }
func (*recordingBackend) RemoveLinkByType(string, string, string) error              { return nil }
func (*recordingBackend) RemoveLinks(string, []backend.LinkRemoval) error            { return nil }
func (*recordingBackend) Promote(string, time.Time, note.Status) error               { return nil }
func (*recordingBackend) Update(*note.Note, *time.Time) error                        { return nil }
func (*recordingBackend) UpdateLink(string, string, *string, *string, *string) error { return nil }
func (*recordingBackend) BulkUpdateLinks(string, []backend.LinkUpdate) error         { return nil }
func (*recordingBackend) BulkWrite([]*note.Note) error                               { return nil }
func (*recordingBackend) BulkApply([]*note.Note, []*note.Note) error                 { return nil }

func TestProductionRootMediaDoctorUsesConcreteService(t *testing.T) {
	_, execute := setupNotebook(t)
	out, err := execute("media", "--json", "doctor")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "media_service_unavailable") {
		t.Fatalf("production media command uses concrete service: %s", out)
	}
}

func TestMediaCommandExposesADR0040Operations(t *testing.T) {
	cmd := newMediaCmd(&rootState{}, nil)
	for _, name := range []string{"inspect", "sample", "transcribe", "prepare", "context", "doctor", "runs"} {
		child, _, err := cmd.Find([]string{name})
		if err != nil || child == cmd || child.Name() != name {
			t.Errorf("media command exposes %q subcommand: missing", name)
		}
	}
}

func TestMediaJSONRendersVersionedEnvelope(t *testing.T) {
	service := &fakeMediaService{result: mediaCommandResult{SchemaVersion: mediaResultVersion, Command: "inspect", RunID: "run-1", ManifestLocator: "/bundle/manifest.json", Outcome: "succeeded", Stages: []mediaStageResult{{Name: "inspect", Requested: true, State: "succeeded"}}}}
	cmd := newMediaCmd(&rootState{}, service)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--json", "inspect", "movie.mp4"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	var got mediaCommandResult
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("media JSON is one result envelope: %v\n%s", err, out.String())
	}
	if got.SchemaVersion != mediaResultVersion || got.RunID != "run-1" || got.ManifestLocator == "" {
		t.Fatalf("media JSON preserves result fields: %#v", got)
	}
}

func TestMediaNonTTYPreservesExplicitPlan(t *testing.T) {
	service := &fakeMediaService{result: mediaCommandResult{Outcome: "succeeded"}}
	old := isTTYFn
	isTTYFn = func() bool { return false }
	t.Cleanup(func() { isTTYFn = old })
	cmd := newMediaCmd(&rootState{}, service)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetArgs([]string{"transcribe", "movie.mp4", "--engine", "parakeet"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !service.request.NonInteractive || service.request.Engine != "parakeet" {
		t.Fatalf("non-TTY preserves explicit plan: %#v", service.request)
	}
}

func TestRootRegistersMediaCommand(t *testing.T) {
	root := NewRootCmd("")
	child, _, err := root.Find([]string{"media"})
	if err != nil || child == root || child.Name() != "media" {
		t.Fatal("root registers media command: missing")
	}
}

func TestMediaDoctorNonTTYRequiresExplicitConfirmation(t *testing.T) {
	service := &fakeMediaService{result: mediaCommandResult{Outcome: "succeeded"}}
	old := isTTYFn
	isTTYFn = func() bool { return false }
	t.Cleanup(func() { isTTYFn = old })
	cmd := newMediaCmd(&rootState{}, service)
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	cmd.SetArgs([]string{"doctor", "--install", "ffmpeg"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "requires --confirm") || service.doctorCalls != 0 {
		t.Fatalf("non-TTY install refuses before service call: err=%v calls=%d", err, service.doctorCalls)
	}
}

func TestAllMediaCommandsLeaveNotebookUnchanged(t *testing.T) {
	cases := [][]string{{"inspect", "movie.mp4"}, {"sample", "movie.mp4"}, {"transcribe", "movie.mp4", "--engine", "parakeet"}, {"prepare", "movie.mp4", "--frames", "1s"}, {"context", "--run", "run-1"}, {"doctor"}, {"runs", "run-1"}}
	for _, args := range cases {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			store := &recordingBackend{}
			service := &fakeMediaService{result: mediaCommandResult{Outcome: "succeeded", RunID: "run-1"}}
			cmd := newMediaCmd(&rootState{backend: store}, service)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if store.writes != 0 {
				t.Fatalf("media command mutated notebook: args=%v writes=%d", args, store.writes)
			}
		})
	}
}

func TestMediaDocumentationCoversCommandsRemediationRecoveryAndExamples(t *testing.T) {
	paths := []string{filepath.Join("..", "..", "..", "skills", "nn-guide", "SKILL.md"), "show.go"}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, required := range []string{"nn media inspect", "nn media sample", "nn media transcribe", "nn media prepare", "nn media context", "nn media doctor", "nn media runs", "non-TTY", "--confirm", "--run"} {
			if !strings.Contains(text, required) {
				t.Errorf("media documentation %s contains %q", path, required)
			}
		}
	}
}

func TestMediaIntegrateSkillRoutesContextAndImages(t *testing.T) {
	for _, path := range []string{filepath.Join("..", "..", "..", "skills", "nn-navigate", "SKILL.md"), filepath.Join("..", "..", "..", "skills", "nn-navigate", "references", "integrate.md")} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for _, required := range []string{"integrate this media run", "nn media context --run", "image-reading", "transcript", "non-mutating proposal", "not an `nn integrate`"} {
			if !strings.Contains(text, required) {
				t.Errorf("media Integrate routing %s contains %q", path, required)
			}
		}
	}
}

func TestMediaCaptureCommandIsAbsent(t *testing.T) {
	cmd := newMediaCmd(&rootState{}, &fakeMediaService{})
	child, _, err := cmd.Find([]string{"capture"})
	if err == nil || child != cmd {
		t.Fatalf("capture must not resolve as a media subcommand: child=%q err=%v", child.Name(), err)
	}
}

func TestMediaJSONOmitsNoteCapture(t *testing.T) {
	service := &fakeMediaService{result: mediaCommandResult{SchemaVersion: mediaResultVersion, Command: "prepare", Outcome: "succeeded"}}
	cmd := newMediaCmd(&rootState{}, service)
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--json", "prepare", "movie.mp4"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "note_capture") {
		t.Fatalf("media result exposes removed note_capture state: %s", out.String())
	}
}
