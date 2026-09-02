package media

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestContractSymbolsMarshalStably(t *testing.T) {
	result := RunResult{SchemaVersion: RunResultSchemaVersion, Outcome: OutcomePartial}
	got, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"schema_version":"nn.media.run-result/v1","outcome":"partial"}`
	if string(got) != want {
		t.Fatalf("stable schema assertion: got %s want %s", got, want)
	}

	manifest := Manifest{SchemaVersion: ManifestSchemaVersion, Lifecycle: LifecycleStaging}
	got, err = json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	want = `{"schema_version":"nn.media.manifest/v1","lifecycle":"staging"}`
	if string(got) != want {
		t.Fatalf("stable manifest assertion: got %s want %s", got, want)
	}
}

func TestDeriveOutcome(t *testing.T) {
	tests := []struct {
		name   string
		stages []StageResult
		note   StageState
		want   Outcome
	}{
		{"success", []StageResult{{Requested: true, State: StageSucceeded}}, StageNotRequested, OutcomeSucceeded},
		{"partial", []StageResult{{Requested: true, State: StageSucceeded}, {Requested: true, State: StageFailed}}, StageNotRequested, OutcomePartial},
		{"prerequisite", []StageResult{{Requested: true, State: StageUnavailable}}, StageNotRequested, OutcomePrerequisiteFailed},
		{"processing", []StageResult{{Requested: true, State: StageFailed}}, StageNotRequested, OutcomeProcessingFailed},
		{"cancelled", []StageResult{{Requested: true, State: StageCancelled}}, StageNotRequested, OutcomeCancelled},
		{"note capture", []StageResult{{Requested: true, State: StageSucceeded}}, StageFailed, OutcomeNoteCaptureFailed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeriveOutcome(tt.stages, tt.note); got != tt.want {
				t.Fatalf("derived outcome assertion: got %q want %q", got, tt.want)
			}
		})
	}
}

func TestLifecycleTransitions(t *testing.T) {
	if !CanTransitionStage(StagePending, StageRunning) || !CanTransitionStage(StageRunning, StagePartial) {
		t.Fatal("legal stage transition assertion: expected pending->running->partial")
	}
	if CanTransitionStage(StageSucceeded, StageRunning) {
		t.Fatal("terminal stage transition assertion: succeeded must not return to running")
	}
	if !CanTransitionLifecycle(LifecycleStaging, LifecycleCompleted) || !CanTransitionLifecycle(LifecycleCompleted, LifecycleSuperseded) {
		t.Fatal("legal bundle transition assertion: expected staging->completed->superseded")
	}
	if CanTransitionLifecycle(LifecycleDeleted, LifecycleCompleted) {
		t.Fatal("terminal bundle transition assertion: deleted must be terminal")
	}
}

func TestManifestValidation(t *testing.T) {
	now := time.Unix(1, 0).UTC()
	m := Manifest{
		SchemaVersion: ManifestSchemaVersion,
		RunID:         "run-1", BundleID: "bundle-1", SourceID: "source-1",
		Lifecycle: LifecycleCompleted,
		Evidence: []EvidenceRecord{
			{ID: "source", Kind: EvidenceSourceIdentity},
			{ID: "frame", Kind: EvidenceExtractedArtifact, Inputs: []string{"source"}, Transformation: "frame extraction"},
			{ID: "claim", Kind: EvidenceAuthoredObservation, Supports: []string{"frame"}, Authorship: &Authorship{Author: "tester", CreatedAt: now, Method: "inspection"}},
		},
	}
	if err := m.Validate(); err != nil {
		t.Fatalf("valid provenance assertion: %v", err)
	}
	m.Evidence[1].Inputs = nil
	if err := m.Validate(); err == nil {
		t.Fatal("derivative provenance assertion: missing immediate input accepted")
	}
}

func TestCoverageKeepsQualifiedStatesDistinct(t *testing.T) {
	coverage := Coverage{
		Transcript: TranscriptCoverage{Attempted: TimeSpan{Start: 0, End: 10 * time.Second}, Represented: []TimeSpan{{Start: 0, End: 8 * time.Second}}, Completion: KnowledgeFailedToDetermine, Confidence: KnowledgeUnknown},
		Video:      VideoCoverage{SampledInstants: []MediaTimestamp{{Timeline: "video", TimeBase: "1/1000", Requested: time.Second, Effective: 1100 * time.Millisecond}}},
		KnownGaps:  []CoverageGap{{Span: TimeSpan{Start: 8 * time.Second, End: 10 * time.Second}, State: KnowledgeKnownAbsent}},
	}
	if coverage.Transcript.Completion == coverage.Transcript.Confidence || coverage.KnownGaps[0].State == KnowledgeUnknown {
		t.Fatal("qualified coverage assertion: distinct states collapsed")
	}
	if len(coverage.Video.SampledInstants) != 1 {
		t.Fatal("sampled instant assertion: video sampling must record instants")
	}
}

func TestRunnerContract(t *testing.T) {
	var _ ProcessRunner = runnerStub{}
	request := CommandSpec{Executable: "ffprobe", Args: []string{"input"}, Environment: map[string]string{"PATH": "/bin"}, WorkingDirectory: "/tmp", Deadline: time.Second, LogLimitBytes: 1024, ResourcePolicy: ResourcePolicy{MaxProcesses: 1}}
	result, _ := runnerStub{}.Run(context.Background(), request)
	if !result.Started || result.ExitCode != 0 || len(result.ArtifactValidations) != 1 {
		t.Fatal("runner result assertion: structured execution fields unavailable")
	}
}

func TestCapabilityProviderContract(t *testing.T) {
	var _ CapabilityProvider = capabilityStub{}
	got, _ := capabilityStub{}.Discover(context.Background(), CapabilityRequest{Operation: OperationTranscribe, AllowSideEffects: false})
	if got.Operation != OperationTranscribe || got.Executable.Path == "" || !got.SideEffectBounded {
		t.Fatal("capability assertion: operation-specific bounded discovery unavailable")
	}
}

func TestTranscriptionAdapterContract(t *testing.T) {
	var _ TranscriptionAdapter = adapterStub{}
	decl := adapterStub{}.Declaration()
	if len(decl.InputModes) == 0 || decl.TimestampOrigin == "" || decl.TrustClass == "" {
		t.Fatal("adapter declaration assertion: required semantics unavailable")
	}
	got, _ := adapterStub{}.Transcribe(context.Background(), TranscriptionRequest{InputPath: "audio.wav"})
	if len(got.Segments) != 1 || got.EffectiveConfig["token"] != RedactedValue {
		t.Fatal("adapter result assertion: timestamped result or redaction unavailable")
	}
}

func TestIdentityAndLifecycleDomainsRemainDistinct(t *testing.T) {
	m := Manifest{RunID: RunID("run"), BundleID: BundleID("bundle"), SourceID: SourceID("source"), Custody: CustodyReferenceInPlace, Publication: PublicationPublished, Lineage: []RunLineage{{Relation: LineageRetryOf, RunID: RunID("old")}}}
	if string(m.RunID) == string(m.BundleID) || m.Custody == "" || m.Publication == "" || len(m.Lineage) != 1 {
		t.Fatal("identity domain assertion: distinct identities or lifecycle domains unavailable")
	}
}

type runnerStub struct{}

func (runnerStub) Run(context.Context, CommandSpec) (CommandResult, error) {
	return CommandResult{Started: true, ArtifactValidations: []ArtifactValidation{{Path: "out", Valid: true}}}, nil
}

type capabilityStub struct{}

func (capabilityStub) Discover(context.Context, CapabilityRequest) (CapabilityReport, error) {
	return CapabilityReport{Operation: OperationTranscribe, Executable: ExecutableIdentity{Path: "/bin/engine"}, SideEffectBounded: true}, nil
}

type adapterStub struct{}

func (adapterStub) Declaration() AdapterDeclaration {
	return AdapterDeclaration{InputModes: []InputMode{InputPreparedAudio}, TimestampOrigin: TimestampMediaStart, TrustClass: TrustExperimentalLocal}
}
func (adapterStub) Transcribe(context.Context, TranscriptionRequest) (TranscriptionResult, error) {
	return TranscriptionResult{Segments: []TranscriptSegment{{Start: 0, End: time.Second, Text: "hello"}}, EffectiveConfig: map[string]string{"token": RedactedValue}}, nil
}
